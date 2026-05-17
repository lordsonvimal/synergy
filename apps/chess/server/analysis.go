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
		return
	}
	hub := g.Hub
	runner.Start(g.ID, g.ID, fen, whiteToMove, func(snap analysis.EvalSnapshot) {
		hub.BroadcastSignals(snapshotToSignals(snap))
	})
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
	if _, cached := runner.LatestForFEN(fen); cached {
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
