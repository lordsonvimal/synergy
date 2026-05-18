package server

import (
	"context"
	"encoding/json"

	"github.com/lordsonvimal/synergy/apps/chess/analysis"
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/ui/ui_store"
)

// triggerSoloAnalysis kicks off (or restarts) the live engine analysis for a
// solo game's current position and forwards EvalSnapshots to all hub
// subscribers. No-op if analysis is unconfigured, the game is over, or the
// game is not solo. The jobKey is the gameID so a new move cancels the prior
// in-flight analysis.
func triggerSoloAnalysis(ctx context.Context, g *game.Game, fen string, whiteToMove bool) {
	if g.PlayMeta != nil {
		return
	}
	runner, ok := analysis.FromContext(ctx)
	if !ok {
		return
	}
	if !g.IsOngoing() {
		// Game just ended — engine search is moot but the eval bar is still
		// showing the pre-terminal eval. Synthesize a definitive terminal
		// snapshot from game state, record it (so LatestForGame stays
		// correct for late connects), and broadcast it via the hub.
		snap := terminalEvalSnapshot(g)
		runner.RecordTerminal(g.ID, fen, snap)
		g.Hub.BroadcastSignals(snapshotToSignals(snap))
		return
	}
	hub := g.Hub
	runner.Start(g.ID, g.ID, fen, whiteToMove, func(snap analysis.EvalSnapshot) {
		hub.BroadcastSignals(snapshotToSignals(snap))
	})
}

// terminalEvalSnapshot builds an EvalSnapshot for a finished game. The result
// is derived entirely from game state — no engine call needed — and uses
// chess result notation (1-0 / 0-1 / ½-½) so the bar reads as a final
// outcome rather than a centipawn score.
func terminalEvalSnapshot(g *game.Game) analysis.EvalSnapshot {
	snap := analysis.EvalSnapshot{EvalReady: true}
	switch g.Winner {
	case engine.White:
		snap.EvalWp = 100
		snap.EvalBlackHeight = "0%"
		snap.EvalTextColor = "#404040"
		snap.EvalDisplay = "1-0"
	case engine.Black:
		snap.EvalWp = 0
		snap.EvalBlackHeight = "100%"
		snap.EvalTextColor = "#f5f5f5"
		snap.EvalDisplay = "0-1"
	default:
		snap.EvalWp = 50
		snap.EvalBlackHeight = "50%"
		snap.EvalTextColor = "#404040"
		snap.EvalDisplay = "½-½"
	}
	return snap
}

// triggerHistoryAnalysis kicks off analysis for a historical position. It uses
// a separate cancellation slot ("hist:"+gameID) so it does not preempt the
// live job, and tags snapshots with no gameID so the live broadcast path
// continues to see the live position's last evaluation. Updates stream to all
// hub subscribers — when the user is viewing this exact position, the bar
// updates; when they navigate away, the next history trigger cancels this
// one.
func triggerHistoryAnalysis(ctx context.Context, gameID, fen string, whiteToMove bool, hub *game.GameHub) {
	runner, ok := analysis.FromContext(ctx)
	if !ok {
		return
	}
	// Re-run if the cached entry never reached a scored update (e.g. a prior
	// job was cancelled before its first depth tick, or pool.Acquire failed).
	// Otherwise the FEN would stay stuck with EvalReady=false forever.
	if snap, cached := runner.LatestForFEN(fen); cached && snap.EvalReady {
		return
	}
	runner.Start("hist:"+gameID, "", fen, whiteToMove, func(snap analysis.EvalSnapshot) {
		hub.BroadcastSignals(snapshotToSignals(snap))
	})
}

// seedSoloAnalysisOnConnect pushes the cached eval snapshot (if any) directly
// to a freshly opened SSE connection so the eval bar reflects current state
// immediately rather than after the next analysis update.
func seedSoloAnalysisOnConnect(ctx context.Context, gameID string, write func(map[string]any)) {
	runner, ok := analysis.FromContext(ctx)
	if !ok {
		return
	}
	snap, ok := runner.LatestForGame(gameID)
	if !ok {
		return
	}
	write(snapshotToSignals(snap))
}

// copyEvalIntoSignals populates the eval fields on signals from the runner's
// most recent snapshot for gameID. This is what prevents the eval bar from
// snapping back to the 50/50 default on every move broadcast (and stops the
// bar from resetting when the game ends, since no further analysis updates
// will follow).
func copyEvalIntoSignals(ctx context.Context, gameID string, signals *ui_store.ChessBoardSignals) {
	runner, ok := analysis.FromContext(ctx)
	if !ok {
		return
	}
	snap, ok := runner.LatestForGame(gameID)
	if !ok {
		return
	}
	signals.EvalCp = snap.EvalCp
	signals.EvalMate = snap.EvalMate
	signals.EvalDepth = snap.EvalDepth
	signals.EvalWp = snap.EvalWp
	signals.EvalDisplay = snap.EvalDisplay
	signals.EvalBlackHeight = snap.EvalBlackHeight
	signals.EvalTextColor = snap.EvalTextColor
	signals.EvalReady = snap.EvalReady
	signals.EvalAnalyzing = snap.EvalAnalyzing
}

// cancelHistoryAnalysis stops any in-flight history-navigation analysis for
// gameID. Called when the user returns to the live position so the lingering
// history job can't keep broadcasting eval frames that overwrite the live
// eval on the client.
func cancelHistoryAnalysis(ctx context.Context, gameID string) {
	runner, ok := analysis.FromContext(ctx)
	if !ok {
		return
	}
	runner.CancelHistory(gameID)
}

// pushEvalForFenDirect writes the cached eval snapshot for fen (if any)
// directly to one SSE connection. Used by history navigation handlers so the
// eval bar matches the displayed position immediately.
func pushEvalForFenDirect(ctx context.Context, fen string, write func(map[string]any)) {
	runner, ok := analysis.FromContext(ctx)
	if !ok {
		return
	}
	snap, ok := runner.LatestForFEN(fen)
	if !ok {
		return
	}
	write(snapshotToSignals(snap))
}

// broadcastLiveEval pushes the runner's latest live snapshot for gameID
// through the hub so it lands on every subscriber's /events stream in order
// with any other queued eval frames. Used on return-to-live: a per-request
// direct response push could be overtaken on the wire by a hist-eval frame
// still queued in a subscriber's hub channel (CancelHistory drains the
// producer goroutine but not the per-subscriber buffers). Broadcasting via
// the hub puts the live frame strictly after those queued frames on each
// subscriber's channel and also keeps other open tabs of the same game in
// sync. gameLatest is preferred over fenCache lookup-by-FEN because history
// jobs pass gameID="" and so can't pollute it — it's the authoritative
// "last live eval" even if a cache-by-FEN lookup would miss.
func broadcastLiveEval(ctx context.Context, gameID string, hub *game.GameHub) {
	runner, ok := analysis.FromContext(ctx)
	if !ok {
		return
	}
	snap, ok := runner.LatestForGame(gameID)
	if !ok {
		return
	}
	hub.BroadcastSignals(snapshotToSignals(snap))
}

func snapshotToSignals(snap analysis.EvalSnapshot) map[string]any {
	b, err := json.Marshal(snap)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

// whiteToMove returns true when the next move belongs to white.
func whiteToMove(side engine.Color) bool { return side == engine.White }
