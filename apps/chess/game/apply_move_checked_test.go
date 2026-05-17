package game_test

import (
	"sync"
	"testing"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
)

func TestApplyMoveChecked_AppliedAdvancesSeq(t *testing.T) {
	g := newTestGame(t)
	prev := g.Seq
	res, seq := g.ApplyMoveChecked(engine.MoveFromUCI("e2e4"), 0, int64(prev))
	if res != game.MoveApplied {
		t.Fatalf("expected MoveApplied, got %v", res)
	}
	if seq != prev+1 {
		t.Fatalf("expected seq %d, got %d", prev+1, seq)
	}
}

func TestApplyMoveChecked_SeqConflictRejects(t *testing.T) {
	g := newTestGame(t)
	g.ApplyMove(engine.MoveFromUCI("e2e4"), 0) // server advances to Seq 1
	// Client thinks it's still at Seq 0 — must be rejected without mutating
	// the board.
	histBefore := g.HistoryLen()
	res, seq := g.ApplyMoveChecked(engine.MoveFromUCI("e7e5"), 0, 0)
	if res != game.MoveSeqConflict {
		t.Fatalf("expected MoveSeqConflict, got %v", res)
	}
	if seq != 1 {
		t.Fatalf("conflict result should report current seq=1, got %d", seq)
	}
	if g.HistoryLen() != histBefore {
		t.Fatalf("board mutated on conflict: history went %d -> %d", histBefore, g.HistoryLen())
	}
}

func TestApplyMoveChecked_NegativeBaseSeqSkipsCheck(t *testing.T) {
	g := newTestGame(t)
	g.ApplyMove(engine.MoveFromUCI("e2e4"), 0) // Seq=1
	// Solo / legacy callers pass baseSeq=-1 to opt out.
	res, _ := g.ApplyMoveChecked(engine.MoveFromUCI("e7e5"), 0, -1)
	if res != game.MoveApplied {
		t.Fatalf("expected MoveApplied with baseSeq=-1, got %v", res)
	}
}

func TestApplyMoveChecked_IllegalMove(t *testing.T) {
	g := newTestGame(t)
	// a1a8 — rook can't leap through pieces from the starting position.
	res, _ := g.ApplyMoveChecked(engine.MoveFromUCI("a1a8"), 0, int64(g.Seq))
	if res != game.MoveIllegal {
		t.Fatalf("expected MoveIllegal for a1-a8 from start (blocked), got %v", res)
	}
}

func TestApplyMoveChecked_PromoNeeded(t *testing.T) {
	g := newTestGame(t)
	// Set up a position with a white pawn on b7 and the b8 square empty by
	// loading a custom FEN directly into the game's board. This avoids a long
	// fragile move sequence and isolates the promotion-gate semantics.
	b, err := engine.BoardFromFEN("r3k3/1P6/8/8/8/8/8/4K3 w - - 0 1")
	if err != nil {
		t.Fatalf("BoardFromFEN: %v", err)
	}
	g.Board = b

	// b7 → a8 capturing rook with no promotion piece set should request promo.
	m := engine.Move{From: 49, To: 56, Promotion: engine.NoPiece}
	res, _ := g.ApplyMoveChecked(m, 0, int64(g.Seq))
	if res != game.MovePromoNeeded {
		t.Fatalf("expected MovePromoNeeded, got %v", res)
	}
	// With a promotion piece set, the move applies.
	m.Promotion = engine.Queen
	res, _ = g.ApplyMoveChecked(m, 0, int64(g.Seq))
	if res != game.MoveApplied {
		t.Fatalf("expected MoveApplied with Queen promotion, got %v", res)
	}
}

// TestApplyMoveChecked_ConcurrentSeqGuard verifies that when two goroutines
// race to apply a move with the same baseSeq, exactly one wins. This is the
// core invariant protecting against same-client out-of-order delivery on the
// server: even if HTTP/2 reorders two POSTs into the handler concurrently,
// only one is accepted; the loser sees MoveSeqConflict and the client can
// reconcile.
func TestApplyMoveChecked_ConcurrentSeqGuard(t *testing.T) {
	g := newTestGame(t)
	const N = 16
	var wg sync.WaitGroup
	var appliedCount, conflictCount int
	var mu sync.Mutex

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, _ := g.ApplyMoveChecked(engine.MoveFromUCI("e2e4"), 0, 0)
			mu.Lock()
			switch res {
			case game.MoveApplied:
				appliedCount++
			case game.MoveSeqConflict:
				conflictCount++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if appliedCount != 1 {
		t.Fatalf("exactly one goroutine should apply, got applied=%d conflict=%d", appliedCount, conflictCount)
	}
	if appliedCount+conflictCount != N {
		t.Fatalf("all results must be applied or conflict, got applied=%d conflict=%d total=%d", appliedCount, conflictCount, N)
	}
	if g.HistoryLen() != 1 {
		t.Fatalf("expected exactly 1 move in history, got %d", g.HistoryLen())
	}
}
