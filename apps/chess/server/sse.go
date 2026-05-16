package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/logger"
	"github.com/lordsonvimal/synergy/apps/chess/ui/components"
	"github.com/lordsonvimal/synergy/apps/chess/ui/ui_store"
	"github.com/starfederation/datastar-go/datastar"
)

// broadcastBoard sends an updated board + notation panel HTML and signals patch
// via DataStar SSE to the requesting player. Route prefix is derived from PlayMeta.
//
// All three frames (board HTML, notation HTML, signals JSON) are buffered into
// one Write so the browser performs at most one layout/paint pass per move
// instead of three back-to-back morphs.
func broadcastBoard(c *gin.Context, g *game.Game, signals *ui_store.ChessBoardSignals) error {
	ctx := c.Request.Context()
	routePrefix := routePrefixFor(g)

	var all []byte
	buf := new(strings.Builder)

	components.RenderChessBoard(g, routePrefix).Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	buf.Reset()
	components.MoveNotationPanel(routePrefix, g.ID, g.History).Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	b, err := json.Marshal(signals)
	if err != nil {
		return err
	}
	all = append(all, datastarPatchSignalsFrame(b)...)

	// Set SSE headers (datastar-go's NewSSE does this; we mirror it manually
	// because we are writing pre-formatted frames).
	h := c.Writer.Header()
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
	}
	if _, err := c.Writer.Write(all); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

// broadcastSignals sends a DataStar signal patch via SSE to the requesting player.
func broadcastSignals(c *gin.Context, signals *ui_store.ChessBoardSignals) error {
	sse := datastar.NewSSE(c.Writer, c.Request)

	b, err := json.Marshal(signals)
	if err != nil {
		return err
	}

	sse.PatchSignals(b)
	return nil
}

// hubBroadcastBoard pushes board + notation + signals as DataStar SSE frames to
// every hub subscriber. Called after a move is applied in play mode so the
// opponent's board updates in real time via the /play/:id/events endpoint.
func hubBroadcastBoard(ctx context.Context, g *game.Game, signals *ui_store.ChessBoardSignals) {
	var all []byte

	buf := new(strings.Builder)
	components.RenderChessBoard(g, "/play").Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	buf.Reset()
	components.MoveNotationPanel("/play", g.ID, g.History).Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	if b, err := json.Marshal(signals); err == nil {
		all = append(all, datastarPatchSignalsFrame(b)...)
	}

	g.Hub.Broadcast(all)
}

// routePrefixFor returns "/play" for online games, "/solo" for solo games.
func routePrefixFor(g *game.Game) string {
	if g.PlayMeta != nil {
		return "/play"
	}
	return "/solo"
}

// datastarPatchElementsFrame formats an HTML string as a DataStar patch-elements SSE frame.
func datastarPatchElementsFrame(html string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "event: %s\n", datastar.EventTypePatchElements)
	for _, line := range strings.Split(html, "\n") {
		fmt.Fprintf(&buf, "data: %s%s\n", datastar.ElementsDatalineLiteral, line)
	}
	buf.WriteByte('\n')
	return buf.Bytes()
}

// datastarPatchSignalsFrame formats JSON as a DataStar patch-signals SSE frame.
func datastarPatchSignalsFrame(b []byte) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "event: %s\n", datastar.EventTypePatchSignals)
	for _, line := range strings.Split(string(b), "\n") {
		fmt.Fprintf(&buf, "data: %s%s\n", datastar.SignalsDatalineLiteral, line)
	}
	buf.WriteByte('\n')
	return buf.Bytes()
}

// applyMoveRequest is the authoritative move handler for both solo and play
// modes. Selection is now driven entirely client-side by chessops (see
// assets/js/board.js); this endpoint receives only an attempted move and
// applies it after re-validating with the Go engine.
//
// Auth, turn enforcement, and game-existence checks are the caller's job.
func applyMoveRequest(c *gin.Context, g *game.Game, from, to uint8, promo engine.Piece) {
	ctx := c.Request.Context()

	signals := ui_store.NewChessBoardSignals()
	datastar.ReadSignals(c.Request, signals)

	// clientTsNs may also arrive in the query string for plain fetch() POSTs
	// from board.js (which can't easily set the Datastar signals body).
	if signals.ClientTsNs == 0 {
		if v := c.Query("clientTsNs"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				signals.ClientTsNs = n
			}
		}
	}

	// One-way network lag → clock compensation for online play.
	lagCompNs := int64(0)
	if g.PlayMeta != nil && signals.ClientTsNs > 0 {
		const maxLagNs = 200_000_000 // 200 ms ceiling
		if lag := time.Now().UnixNano() - signals.ClientTsNs; lag > 0 && lag < maxLagNs {
			lagCompNs = lag
		}
	}

	move := engine.Move{From: from, To: to, Promotion: promo}

	// Server-side selection state has to be in sync with the attempted move
	// or ApplyMove will reject. Plant a temporary selection from the move's
	// source so the existing IsTarget/IsPromotionMove checks keep working.
	g.SelectSquare(ctx, from)
	if !g.HasSelection() || g.GetSelectionFrom() != from || !g.IsTarget(to) {
		// Illegal move: clear selection, signal the rejection by pushing the
		// authoritative state so the client can hard-reset its DOM.
		g.ClearSelection()
		signals.UpdateFromGame(g)
		_ = broadcastBoard(c, g, signals)
		c.Status(http.StatusOK)
		return
	}

	if g.IsPromotionMove(move) && promo == engine.NoPiece {
		// Client should have settled promotion before POSTing. Treat as
		// rejection and let the client reconcile from the morph.
		g.ClearSelection()
		signals.UpdateFromGame(g)
		_ = broadcastBoard(c, g, signals)
		return
	}

	if !g.ApplyMove(move, lagCompNs) {
		g.ClearSelection()
		signals.UpdateFromGame(g)
		_ = broadcastBoard(c, g, signals)
		return
	}

	signals.UpdateFromGame(g)
	signals.ClearPromotion()
	if err := broadcastBoard(c, g, signals); err != nil {
		logger.Error(ctx).Err(err).Msg("applyMoveRequest: broadcast board")
	}
	if g.PlayMeta != nil {
		hubBroadcastBoard(ctx, g, signals)
	}
}

// writeClockSnapshot writes a single datastar signal patch carrying the given
// clock snapshot directly to this connection (not via the hub).
func writeClockSnapshot(c *gin.Context, snap game.ClockTickEvent) {
	writeSignalsDirect(c, map[string]any{
		"clkW":    snap.WhiteRemainingNs,
		"clkB":    snap.BlackRemainingNs,
		"clkWRun": snap.WhiteRunning,
		"clkBRun": snap.BlackRunning,
		"clkTs":   snap.ServerTsNs,
	})
}

// writeSignalsDirect writes a datastar signal patch directly to one connection,
// bypassing the hub. Used for connection-specific initial state.
func writeSignalsDirect(c *gin.Context, signals map[string]any) {
	b, err := json.Marshal(signals)
	if err != nil {
		return
	}
	c.Writer.Write(datastarPatchSignalsFrame(b))
	c.Writer.Flush()
}

// syncDatastarBoard pushes the current board state to a Datastar SSE connection.
// Called on every new Datastar connection so reconnects recover any missed move.
func syncDatastarBoard(ctx context.Context, c *gin.Context, g *game.Game, role string) {
	var all []byte

	buf := new(strings.Builder)
	components.RenderChessBoard(g, "/play").Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	buf.Reset()
	components.MoveNotationPanel("/play", g.ID, g.History).Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	signals := ui_store.ChessBoardSignalsFromGame(g)
	// Don't replay the side-to-move's pending selection to the opponent on
	// (re)connect — it's their private pre-move state.
	if !ui_store.RoleMatchesSide(role, signals.SideToMove) {
		signals.ClearSelection()
	}
	if b, err := json.Marshal(signals); err == nil {
		all = append(all, datastarPatchSignalsFrame(b)...)
	}

	c.Writer.Write(all)
	c.Writer.Flush()
}
