package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
//
// The serverSelectionSeq field is intentionally stripped from the broadcast:
// it is a per-mover acknowledgement and would otherwise advance the opponent's
// $ackSelectionSeq past their own $selectionSeq, breaking the freshness gate
// on the opponent's board (selection highlights would freeze on their previous
// pending state and they'd have to click many times before highlights worked
// again). The mover still receives the seq via the POST response.
func hubBroadcastBoard(ctx context.Context, g *game.Game, signals *ui_store.ChessBoardSignals) {
	var all []byte

	buf := new(strings.Builder)
	components.RenderChessBoard(g, "/play").Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	buf.Reset()
	components.MoveNotationPanel("/play", g.ID, g.History).Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	if b, err := marshalSignalsForHub(signals); err == nil {
		all = append(all, datastarPatchSignalsFrame(b)...)
	}

	g.Hub.Broadcast(all)
}

// marshalSignalsForHub marshals signals but drops serverSelectionSeq so the
// per-mover ack doesn't leak to other subscribers via the hub.
func marshalSignalsForHub(signals *ui_store.ChessBoardSignals) ([]byte, error) {
	b, err := json.Marshal(signals)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	delete(m, "serverSelectionSeq")
	return json.Marshal(m)
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

// applySquareSelection processes a square click for both solo and play modes.
// Auth and turn enforcement are the caller's responsibility.
func applySquareSelection(c *gin.Context, g *game.Game, square uint8) {
	ctx := c.Request.Context()

	signals := ui_store.NewChessBoardSignals()
	datastar.ReadSignals(c.Request, signals)

	// Drop stale selections: when the client rapidly fires multiple clicks, only
	// the highest-seq one is processed. Older seqs return empty (no SSE body) so
	// they can never overwrite a newer selection on the wire.
	if !g.TryAdvanceSelectionSeq(signals.SelectionSeq) {
		c.Status(http.StatusNoContent)
		return
	}

	// Compute one-way network lag for clock compensation in online games.
	// Client stamps Date.now()*1e6 into clientTsNs just before the POST fires.
	lagCompNs := int64(0)
	if g.PlayMeta != nil && signals.ClientTsNs > 0 {
		const maxLagNs = 200_000_000 // 200 ms ceiling
		if lag := time.Now().UnixNano() - signals.ClientTsNs; lag > 0 && lag < maxLagNs {
			lagCompNs = lag
		}
	}

	if g.HasSelection() && g.IsTarget(square) {
		move := engine.Move{From: g.GetSelectionFrom(), To: square, Promotion: engine.NoPiece}
		promoteWithPiece := signals.Promotion && signals.PromotionPiece != engine.NoPiece

		if g.IsPromotionMove(move) && !promoteWithPiece {
			signals.EnablePromotion(square)
			if err := broadcastSignals(c, signals); err != nil {
				logger.Error(ctx).Err(err).Msg("applySquareSelection: broadcast promotion")
			}
			return
		}
		if promoteWithPiece {
			move.Promotion = signals.PromotionPiece
		}
		if g.ApplyMove(move, lagCompNs) {
			signals.UpdateFromGame(g)
			if promoteWithPiece {
				signals.ClearPromotion()
			}
			// Always send board + notation + signals together in the POST
			// response. This keeps highlight-clear and piece-move atomic for
			// the mover, eliminating the perceived lag where the selection
			// vanished noticeably before the piece appeared in its new
			// square. The hub broadcast below still fans the same payload
			// to the opponent (and idempotently re-morphs the mover's
			// board, which is a no-op visually).
			if err := broadcastBoard(c, g, signals); err != nil {
				logger.Error(ctx).Err(err).Msg("applySquareSelection: broadcast board")
			}
			if g.PlayMeta != nil {
				hubBroadcastBoard(ctx, g, signals)
			}
			return
		}
	}

	g.SelectSquare(ctx, square)
	signals.UpdateFromGame(g)
	if err := broadcastSignals(c, signals); err != nil {
		logger.Error(ctx).Err(err).Msg("applySquareSelection: broadcast selection")
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
