package game_test

import (
	"testing"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
)

func TestPlayMeta_RememberAndRecallMove(t *testing.T) {
	pm := &game.PlayMeta{}

	snap := game.SignalsSnapshot{Fen: "FEN-AFTER", Seq: 7}
	pm.RememberMove(engine.White, "id-1", game.MoveApplied, snap)

	res, got, ok := pm.RecallMove(engine.White, "id-1")
	if !ok {
		t.Fatal("expected cache hit for matching id")
	}
	if res != game.MoveApplied {
		t.Fatalf("expected MoveApplied, got %v", res)
	}
	if got.Fen != snap.Fen || got.Seq != snap.Seq {
		t.Fatalf("snapshot mismatch: got %+v want %+v", got, snap)
	}
}

func TestPlayMeta_RecallMove_DifferentColorIsMiss(t *testing.T) {
	pm := &game.PlayMeta{}
	pm.RememberMove(engine.White, "id-1", game.MoveApplied, game.SignalsSnapshot{})

	if _, _, ok := pm.RecallMove(engine.Black, "id-1"); ok {
		t.Fatal("a White-cached id must not match a Black recall")
	}
}

func TestPlayMeta_RecallMove_StaleIdIsMiss(t *testing.T) {
	pm := &game.PlayMeta{}
	pm.RememberMove(engine.White, "id-1", game.MoveApplied, game.SignalsSnapshot{})
	// Newer move overwrites the slot — old id is forgotten.
	pm.RememberMove(engine.White, "id-2", game.MoveApplied, game.SignalsSnapshot{})

	if _, _, ok := pm.RecallMove(engine.White, "id-1"); ok {
		t.Fatal("only the latest move id should match — older ids must miss")
	}
}

func TestPlayMeta_EmptyIdSkipsCache(t *testing.T) {
	pm := &game.PlayMeta{}
	pm.RememberMove(engine.White, "", game.MoveApplied, game.SignalsSnapshot{})
	// An empty id is a legacy client that opted out — must never produce a hit.
	if _, _, ok := pm.RecallMove(engine.White, ""); ok {
		t.Fatal("empty id must not match")
	}
}
