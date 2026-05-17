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
// every hub subscriber. Called after every applied move so the always-on
// /events SSE delivers the morph to the moving player (and the opponent, in
// play mode).
func hubBroadcastBoard(ctx context.Context, g *game.Game, signals *ui_store.ChessBoardSignals) {
	routePrefix := routePrefixFor(g)
	var all []byte

	buf := new(strings.Builder)
	components.RenderChessBoard(g, routePrefix).Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	buf.Reset()
	components.MoveNotationPanel(routePrefix, g.ID, g.History).Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	// Re-render the promotion overlay so its buttons reflect the new
	// sideToMove (the player about to make the next move). Without this, a
	// black promotion would still show white piece SVGs from page load.
	buf.Reset()
	components.RenderPromotionOverlay(g, routePrefix).Render(ctx, buf)
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

// callerColor returns the engine.Color of the authenticated play-session
// caller (White / Black). Returns false for solo / unauthenticated /
// spectator callers — those skip the per-color idempotency cache.
func callerColor(c *gin.Context) (engine.Color, bool) {
	claims, err := GetPlaySession(c.Request)
	if err != nil {
		return 0, false
	}
	switch claims.Role {
	case "white":
		return engine.White, true
	case "black":
		return engine.Black, true
	}
	return 0, false
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

	// Optional baseSeq from the client: the Seq value the client believes the
	// game is currently at. -1 = client did not send one (legacy / solo).
	baseSeq := int64(-1)
	if v := c.Query("baseSeq"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			baseSeq = n
		}
	}
	clientMoveId := c.Query("clientMoveId")

	// Idempotency: if this is a retry of a move we've already processed for
	// this color (same clientMoveId), return the cached outcome. Without
	// this, a network-retried POST against an already-applied move would
	// look like a stale-baseSeq conflict and trigger a needless rollback on
	// the client. Solo games have no PlayMeta and skip the cache.
	var cachedResult *game.MoveResult
	var cachedSnap game.SignalsSnapshot
	if g.PlayMeta != nil && clientMoveId != "" {
		// Caller color is required to scope the cache — solo callers don't
		// hit this branch because PlayMeta is nil.
		if color, ok := callerColor(c); ok {
			if res, snap, found := g.PlayMeta.RecallMove(color, clientMoveId); found {
				cachedResult = &res
				cachedSnap = snap
			}
		}
	}

	var result game.MoveResult
	var snap game.SignalsSnapshot
	if cachedResult != nil {
		result = *cachedResult
		snap = cachedSnap
	} else {
		// One-shot apply + validate under a single game lock. The snapshot is
		// captured under the same lock as the apply, so the broadcast we build
		// from it is guaranteed not to mix fields from before and after a
		// concurrent move (Seq=N+1 paired with FEN=N was previously possible).
		result, snap = g.ApplyMoveChecked(move, lagCompNs, baseSeq)
		// Remember the outcome so a network-retried POST is idempotent.
		if g.PlayMeta != nil && clientMoveId != "" {
			if color, ok := callerColor(c); ok {
				g.PlayMeta.RememberMove(color, clientMoveId, result, snap)
			}
		}
	}
	signals.UpdateFromSnapshot(snap)

	// On a cache hit we already broadcast the outcome the first time around
	// and the global state may have moved on since; suppress the rebroadcast
	// so we don't send stale-but-strictly-older frames over the hub. The
	// HTTP status alone is enough for the retrying client.
	skipBroadcast := cachedResult != nil

	// Status MUST be set before broadcastBoard, because broadcastBoard writes
	// to the response body and commits headers — calling c.Status after that
	// silently leaves the status at the default 200.
	switch result {
	case game.MoveApplied:
		c.Status(http.StatusOK)
		if skipBroadcast {
			return
		}
		signals.ClearPromotion()
		if err := broadcastBoard(c, g, signals); err != nil {
			logger.Error(ctx).Err(err).Msg("applyMoveRequest: broadcast board")
		}
		// Push the same frames over the always-on /events SSE. The plain fetch()
		// in board.js drains the POST response without parsing the SSE frames,
		// so without this the move-notation panel and any post-move signals
		// never reach the client in solo mode (and only reach the opponent in
		// play).
		hubBroadcastBoard(ctx, g, signals)
		return

	case game.MoveSeqConflict:
		// Client's view is stale (another move slipped in, or a retried POST
		// after the move already applied). Echo authoritative state so the
		// client resyncs, and tell it explicitly via 409.
		c.Status(http.StatusConflict)
		if skipBroadcast {
			return
		}
		_ = broadcastBoard(c, g, signals)
		hubBroadcastBoard(ctx, g, signals)
		return

	case game.MovePromoNeeded:
		// Pawn reached last rank with no Promotion piece set. Client should
		// open the promotion overlay and re-POST. 422 is the conventional code
		// for "well-formed request, semantically unprocessable".
		c.Status(http.StatusUnprocessableEntity)
		if skipBroadcast {
			return
		}
		_ = broadcastBoard(c, g, signals)
		return

	case game.MoveGameOver:
		c.Status(http.StatusGone)
		if skipBroadcast {
			return
		}
		_ = broadcastBoard(c, g, signals)
		return

	default: // MoveIllegal
		c.Status(http.StatusUnprocessableEntity)
		if skipBroadcast {
			return
		}
		_ = broadcastBoard(c, g, signals)
		return
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
