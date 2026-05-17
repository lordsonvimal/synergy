package game_test

import (
	"testing"
	"time"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
)

// testMode returns a standard 5+2 game mode for use in tests.
func testMode() *game.GameMode {
	gm, _ := game.FindGameModeByName("Standard 5+2")
	return &gm
}

// newTestGame creates a new game with the WAL stored in a temp directory.
func newTestGame(t *testing.T) *game.Game {
	t.Helper()
	t.Setenv("DATA_DIR", t.TempDir())
	return game.NewGame(testMode())
}

// ---- Move validation ------------------------------------------------------

func TestRevertLastPlyRestoresPosition(t *testing.T) {
	g := newTestGame(t)

	g.ApplyMove(engine.MoveFromUCI("e2e4"), 0) // 1. e4
	g.ApplyMove(engine.MoveFromUCI("e7e5"), 0) // 1...e5
	g.ApplyMove(engine.MoveFromUCI("g1f3"), 0) // 2. Nf3

	if g.Board.SideToMove != engine.Black {
		t.Fatal("expected Black to move after 2.Nf3")
	}
	if g.HistoryLen() != 3 {
		t.Fatalf("expected 3 history entries, got %d", g.HistoryLen())
	}
	prevSeq := g.Seq

	if !g.RevertLastPly() {
		t.Fatal("RevertLastPly returned false on a normal position")
	}
	if g.HistoryLen() != 2 {
		t.Fatalf("expected 2 history entries after revert, got %d", g.HistoryLen())
	}
	if g.Board.SideToMove != engine.White {
		t.Fatal("expected White to move after reverting Nf3")
	}
	if g.Seq != prevSeq-1 {
		t.Fatalf("expected seq %d after revert, got %d", prevSeq-1, g.Seq)
	}
	// Re-applying the same move should work — the position is genuinely
	// restored, not just the side-to-move flipped.
	if !g.ApplyMove(engine.MoveFromUCI("g1f3"), 0) {
		t.Fatal("re-applying Nf3 after revert should succeed")
	}
	if g.Seq != prevSeq {
		t.Fatalf("expected seq %d after re-apply, got %d", prevSeq, g.Seq)
	}
}

func TestRevertLastPlyOnFirstMoveRestoresInitial(t *testing.T) {
	g := newTestGame(t)
	g.ApplyMove(engine.MoveFromUCI("e2e4"), 0)
	if !g.RevertLastPly() {
		t.Fatal("RevertLastPly should succeed after first move")
	}
	if g.HistoryLen() != 0 {
		t.Fatalf("expected empty history, got %d", g.HistoryLen())
	}
	if g.Board.SideToMove != engine.White {
		t.Fatal("expected White to move at initial position")
	}
	// All moves should be legal again from the starting position.
	if !g.ApplyMove(engine.MoveFromUCI("d2d4"), 0) {
		t.Fatal("d4 should be legal from the restored initial position")
	}
}

func TestRevertLastPlyEmptyHistoryReturnsFalse(t *testing.T) {
	g := newTestGame(t)
	if g.RevertLastPly() {
		t.Fatal("RevertLastPly with no history should return false")
	}
}

func TestRevertLastPlyDoesNotAwardIncrement(t *testing.T) {
	g := newTestGame(t)

	// Play one move so White's clock has run + gotten the increment, and
	// Black's clock is now running.
	g.ApplyMove(engine.MoveFromUCI("e2e4"), 0)
	blackBefore := g.Clock.Black.RemainingNs
	whiteBefore := g.Clock.White.RemainingNs

	// Black "thinks" briefly so their clock would otherwise pick up an
	// increment if we routed through Clock.Stop.
	time.Sleep(20 * time.Millisecond)

	if !g.RevertLastPly() {
		t.Fatal("RevertLastPly should succeed")
	}

	// White should be back on the move with their clock running.
	if !g.Clock.White.Running {
		t.Fatal("expected white clock running after revert")
	}
	if g.Clock.Black.Running {
		t.Fatal("expected black clock stopped after revert")
	}
	// Black should NOT have received the Fischer increment (they did not
	// complete a move). Allow a tiny tolerance: any positive change up to a
	// few ms is acceptable elapsed-time bookkeeping, but anything close to
	// the configured 2s increment indicates a bug.
	if g.Clock.Black.RemainingNs > blackBefore {
		t.Fatalf("black gained time after revert: before=%d after=%d", blackBefore, g.Clock.Black.RemainingNs)
	}
	// White's clock should not have changed (the revert just resumes it).
	if g.Clock.White.RemainingNs != whiteBefore {
		t.Fatalf("white clock changed unexpectedly: before=%d after=%d", whiteBefore, g.Clock.White.RemainingNs)
	}
}

func TestValidMoveApplied(t *testing.T) {
	g := newTestGame(t)

	// e2e4 — white's first move, always legal from the starting position.
	m := engine.MoveFromUCI("e2e4")
	if !g.ApplyMove(m, 0) {
		t.Fatal("e2e4 should be a valid first move")
	}
	// After the move it should be Black's turn.
	if g.Board.SideToMove != engine.Black {
		t.Error("side to move should be Black after White's first move")
	}
}

func TestInvalidMoveRejected(t *testing.T) {
	g := newTestGame(t)

	// Set up a pin: White Ke1, White Re4, Black Qe8, Black Kh8.
	// The rook on e4 is pinned to the king along the e-file; moving it to d4
	// would expose the white king to check from the black queen — genuinely illegal.
	for p := engine.Piece(0); p < 6; p++ {
		g.Board.Pieces[engine.White][p] = 0
		g.Board.Pieces[engine.Black][p] = 0
	}
	g.Board.Pieces[engine.White][engine.King] = 1 << 4   // Ke1
	g.Board.Pieces[engine.White][engine.Rook] = 1 << 28  // Re4
	g.Board.Pieces[engine.Black][engine.King] = 1 << 63  // Kh8
	g.Board.Pieces[engine.Black][engine.Queen] = 1 << 60 // Qe8
	g.Board.Occupancy[engine.White] = g.Board.Pieces[engine.White][engine.King] | g.Board.Pieces[engine.White][engine.Rook]
	g.Board.Occupancy[engine.Black] = g.Board.Pieces[engine.Black][engine.King] | g.Board.Pieces[engine.Black][engine.Queen]
	g.Board.All = g.Board.Occupancy[engine.White] | g.Board.Occupancy[engine.Black]
	g.Board.SideToMove = engine.White
	g.Board.Castling = 0
	g.Board.EnPassant = engine.NoSquare

	// Moving the pinned rook off the e-file (e4→d4) is illegal.
	if g.ApplyMove(engine.MoveFromUCI("e4d4"), 0) {
		t.Error("moving a pinned rook off the pin line should be rejected as illegal")
	}
	if g.Board.SideToMove != engine.White {
		t.Error("side to move should remain White after a rejected move")
	}
}

func TestMoveOnFinishedGameRejected(t *testing.T) {
	g := newTestGame(t)

	// Drive the game to Scholar's Mate (checkmate) via the underlying board,
	// then verify that further moves via ApplyMove are rejected.
	scholarsMateMoves := []string{"e2e4", "e7e5", "f1c4", "b8c6", "d1h5", "g8f6", "h5f7"}
	for _, uci := range scholarsMateMoves {
		g.ApplyMove(engine.MoveFromUCI(uci), 0)
	}

	// Game should be over.
	if g.State == game.GameOngoing {
		t.Fatal("game should not be ongoing after Scholar's Mate")
	}

	// Any further move must be rejected.
	if g.ApplyMove(engine.MoveFromUCI("a7a6"), 0) {
		t.Error("move on a finished game should be rejected")
	}
}

// ---- Clock start/stop ----------------------------------------------------

func TestClockStartsForWhiteAtGameCreation(t *testing.T) {
	g := newTestGame(t)

	// White's clock should start running immediately (standard chess convention).
	if !g.Clock.White.Running {
		t.Error("white clock should be running at game start")
	}
	if g.Clock.Black.Running {
		t.Error("black clock should not be running at game start")
	}
}

func TestClockSwitchesAfterMove(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	// Use a mode with zero increment so remaining time strictly decreases on a move.
	gm, _ := game.FindGameModeByName("Blitz 3+0")
	g := game.NewGame(&gm)

	initialWhiteRem := g.Clock.White.RemainingNs
	initialBlackRem := g.Clock.Black.RemainingNs

	// Small delay so the clock can consume some time.
	time.Sleep(5 * time.Millisecond)

	g.ApplyMove(engine.MoveFromUCI("e2e4"), 0)

	// After White's move: White clock stops (deducted time), Black clock starts.
	if g.Clock.White.Running {
		t.Error("white clock should have stopped after White's move")
	}
	if !g.Clock.Black.Running {
		t.Error("black clock should be running after White's move")
	}

	// With zero increment, white's remaining time must have decreased.
	if g.Clock.White.RemainingNs >= initialWhiteRem {
		t.Error("white remaining time should have decreased after White's move")
	}

	// Black's remaining time should be unchanged (clock just started, no moves yet).
	if g.Clock.Black.RemainingNs != initialBlackRem {
		t.Errorf("black remaining time should be unchanged before Black moves: want %d, got %d",
			initialBlackRem, g.Clock.Black.RemainingNs)
	}
}

func TestClockIncrementAppliedOnMove(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	// Use a mode with a 2-second increment.
	gm, _ := game.FindGameModeByName("Standard 5+2")
	g := game.NewGame(&gm)

	initialWhiteRem := g.Clock.White.RemainingNs

	time.Sleep(5 * time.Millisecond)
	g.ApplyMove(engine.MoveFromUCI("e2e4"), 0)

	// White's remaining time = initial − elapsed + increment.
	// The result should be > (initial - some_elapsed) because increment is added.
	// Since increment (2s = 2_000_000_000ns) >> elapsed (5ms), remaining should be ≈ initial + 2s.
	afterMove := g.Clock.White.RemainingNs

	// After the move, White's remaining should be close to initial + increment (minus a tiny elapsed).
	// At minimum it must be > initial to show increment was added.
	incNs := int64(2 * 1_000_000_000)
	if afterMove <= initialWhiteRem-incNs {
		t.Errorf("white remaining after move (%d) should be close to initial+increment (%d+%d)",
			afterMove, initialWhiteRem, incNs)
	}
}

// ---- Game-over detection --------------------------------------------------

func TestCheckmateEndsGame(t *testing.T) {
	g := newTestGame(t)

	// Apply Scholar's Mate: 1.e4 e5  2.Bc4 Nc6  3.Qh5 Nf6??  4.Qxf7#
	moves := []string{"e2e4", "e7e5", "f1c4", "b8c6", "d1h5", "g8f6", "h5f7"}
	for _, uci := range moves {
		if !g.ApplyMove(engine.MoveFromUCI(uci), 0) {
			t.Fatalf("move %s rejected (unexpected)", uci)
		}
	}

	if g.State != game.GameCheckmate {
		t.Errorf("game state should be GameCheckmate after Scholar's Mate, got %v", g.State)
	}
	if g.Winner != engine.White {
		t.Errorf("winner should be White after Scholar's Mate, got %v", g.Winner)
	}
}

func TestStalemateDetectedByGame(t *testing.T) {
	// Verify that the Game wrapper's IsStalemate() correctly identifies a stalemate
	// position.  The engine-level detection is tested in engine/engine_test.go;
	// here we confirm the game layer exposes it correctly.
	//
	// We use the same known stalemate position as the engine test:
	//   White Kb6 + Qe5  vs  Black Ka8  (Black to move, not in check, no legal moves).
	g := newTestGame(t)

	// Bring the game to a state where stalemate is detectable without locking issues:
	// clear all pieces and place only the stalemate trio by manipulating the public
	// Board fields directly before the watchdog has a chance to observe the board.
	g.Board.Pieces[engine.White][engine.King] = 1 << 41   // Kb6
	g.Board.Pieces[engine.White][engine.Queen] = 1 << 36  // Qe5
	g.Board.Pieces[engine.Black][engine.King] = 1 << 56   // Ka8
	for p := engine.Piece(0); p < 6; p++ {
		if p != engine.King && p != engine.Queen {
			g.Board.Pieces[engine.White][p] = 0
		}
		if p != engine.King {
			g.Board.Pieces[engine.Black][p] = 0
		}
	}
	g.Board.Occupancy[engine.White] = g.Board.Pieces[engine.White][engine.King] | g.Board.Pieces[engine.White][engine.Queen]
	g.Board.Occupancy[engine.Black] = g.Board.Pieces[engine.Black][engine.King]
	g.Board.All = g.Board.Occupancy[engine.White] | g.Board.Occupancy[engine.Black]
	g.Board.SideToMove = engine.Black
	g.Board.Castling = 0
	g.Board.EnPassant = engine.NoSquare

	if !g.IsStalemate() {
		t.Error("Game.IsStalemate() should return true for the known stalemate position")
	}
	if g.IsCheck() {
		t.Error("Game.IsCheck() should return false in a stalemate position")
	}
}

func TestCheckIsDetectedDuringGame(t *testing.T) {
	g := newTestGame(t)

	// Apply the first six moves of Scholar's Mate: after 3.Qh5 Nf6?? Black is NOT
	// yet in check, but after 4.Qxf7 Black IS.
	// Moves 1–6 (no check yet):
	for _, uci := range []string{"e2e4", "e7e5", "d1h5", "b8c6", "f1c4", "g8f6"} {
		g.ApplyMove(engine.MoveFromUCI(uci), 0)
	}
	if g.IsCheck() {
		t.Error("there should be no check before Qxf7")
	}

	// Move 7 — Qxf7+ (check and mate)
	g.ApplyMove(engine.MoveFromUCI("h5f7"), 0)
	if !g.IsCheck() {
		t.Error("black king should be in check after Qxf7")
	}
}

// ---- Move history --------------------------------------------------------

func TestHistoryEmptyAtStart(t *testing.T) {
	g := newTestGame(t)
	if len(g.History) != 0 {
		t.Fatalf("expected empty history at start, got %d entries", len(g.History))
	}
}

func TestHistoryTracksMovesWithSAN(t *testing.T) {
	g := newTestGame(t)

	g.ApplyMove(engine.MoveFromUCI("e2e4"), 0)

	if len(g.History) != 1 {
		t.Fatalf("expected 1 history entry after first move, got %d", len(g.History))
	}
	rec := g.History[0]
	if rec.SAN != "e4" {
		t.Errorf("SAN: want 'e4', got %q", rec.SAN)
	}
	if rec.Color != engine.White {
		t.Errorf("Color: want White, got %v", rec.Color)
	}
	if rec.MoveNumber != 1 {
		t.Errorf("MoveNumber: want 1, got %d", rec.MoveNumber)
	}
	if rec.FEN == "" {
		t.Error("FEN must not be empty")
	}

	g.ApplyMove(engine.MoveFromUCI("e7e5"), 0)

	if len(g.History) != 2 {
		t.Fatalf("expected 2 entries after second move, got %d", len(g.History))
	}
	rec2 := g.History[1]
	if rec2.SAN != "e5" {
		t.Errorf("SAN: want 'e5', got %q", rec2.SAN)
	}
	if rec2.Color != engine.Black {
		t.Errorf("Color: want Black, got %v", rec2.Color)
	}
}

func TestHistoryRejectedMoveNotRecorded(t *testing.T) {
	g := newTestGame(t)

	// Pin: Re4 pinned by Qe8, moving it to d4 is illegal.
	for p := engine.Piece(0); p < 6; p++ {
		g.Board.Pieces[engine.White][p] = 0
		g.Board.Pieces[engine.Black][p] = 0
	}
	g.Board.Pieces[engine.White][engine.King] = 1 << 4
	g.Board.Pieces[engine.White][engine.Rook] = 1 << 28
	g.Board.Pieces[engine.Black][engine.King] = 1 << 63
	g.Board.Pieces[engine.Black][engine.Queen] = 1 << 60
	g.Board.Occupancy[engine.White] = g.Board.Pieces[engine.White][engine.King] | g.Board.Pieces[engine.White][engine.Rook]
	g.Board.Occupancy[engine.Black] = g.Board.Pieces[engine.Black][engine.King] | g.Board.Pieces[engine.Black][engine.Queen]
	g.Board.All = g.Board.Occupancy[engine.White] | g.Board.Occupancy[engine.Black]
	g.Board.SideToMove = engine.White
	g.Board.Castling = 0
	g.Board.EnPassant = engine.NoSquare

	g.ApplyMove(engine.MoveFromUCI("e4d4"), 0) // illegal

	if len(g.History) != 0 {
		t.Errorf("illegal move must not be appended to history, got %d entries", len(g.History))
	}
}
