package game_test

import (
	"testing"
	"time"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
)

// newTestPlayMeta builds a PlayMeta seeded with the fields that callers in
// the server package set on real game creation, plus an optional join
// deadline override.
func newTestPlayMeta(joinDeadline *time.Time) *game.PlayMeta {
	return &game.PlayMeta{
		WhiteParticipantID: "wp",
		BlackParticipantID: "bp",
		WhiteToken:         "white-token",
		BlackToken:         "black-token",
		WhiteClaimed:       true,
		BlackClaimed:       false,
		RematchProposedBy:  engine.NoColor,
		DrawOfferedBy:      engine.NoColor,
		TakebackOfferedBy:  engine.NoColor,
		LastDrawSeq:        [2]int64{-1, -1},
		LastTakebackSeq:    [2]int64{-1, -1},
		ClaimVictoryFor:    engine.NoColor,
		JoinDeadline:       joinDeadline,
	}
}

// waitUntil polls until cond returns true or the deadline fires.
func waitUntil(cond func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}

// ---- IsSeatClaimed --------------------------------------------------------

func TestIsSeatClaimed(t *testing.T) {
	pm := newTestPlayMeta(nil)
	if !pm.IsSeatClaimed("white") {
		t.Error("white seeded as claimed should report claimed")
	}
	if pm.IsSeatClaimed("black") {
		t.Error("black not seeded as claimed should report unclaimed")
	}
	if pm.IsSeatClaimed("spectator") {
		t.Error("spectator is not a seat — must return false")
	}
	if pm.IsSeatClaimed("") {
		t.Error("empty role must return false")
	}
}

// ---- TryClaim atomicity ---------------------------------------------------

func TestTryClaimSucceedsOnceThenFails(t *testing.T) {
	pm := newTestPlayMeta(nil)

	pid, role, ok := pm.TryClaim("black-token")
	if !ok || role != "black" || pid != "bp" {
		t.Fatalf("first TryClaim: got (%q,%q,%v), want (\"bp\",\"black\",true)", pid, role, ok)
	}
	if !pm.IsSeatClaimed("black") {
		t.Fatal("black seat should be claimed after TryClaim")
	}

	// Second attempt with the same token must fail — protects against
	// duplicate joins from a second visitor.
	if _, _, ok := pm.TryClaim("black-token"); ok {
		t.Error("second TryClaim with same token must fail")
	}
	// Unknown tokens never claim.
	if _, _, ok := pm.TryClaim("garbage"); ok {
		t.Error("TryClaim with unknown token must fail")
	}
}

// ---- RecordSSEConnect clears JoinDeadline --------------------------------

func TestRecordSSEConnectBothPlayersClearsJoinDeadline(t *testing.T) {
	dl := time.Now().Add(60 * time.Second)
	pm := newTestPlayMeta(&dl)

	// One side connecting is not enough.
	if both, _ := pm.RecordSSEConnect("white"); both {
		t.Fatal("bothFirstConnected should be false when only white connects")
	}
	if pm.GetJoinDeadlineNs() == 0 {
		t.Error("JoinDeadline should still be set with only white connected")
	}

	// Second side connecting flips the gate and clears the deadline.
	if both, _ := pm.RecordSSEConnect("black"); !both {
		t.Fatal("bothFirstConnected should be true when black joins white")
	}
	if pm.GetJoinDeadlineNs() != 0 {
		t.Error("JoinDeadline should be cleared once both players connect")
	}

	// Subsequent disconnect/reconnects don't resurrect the deadline.
	pm.RecordSSEDisconnect("black")
	pm.RecordSSEConnect("black")
	if pm.GetJoinDeadlineNs() != 0 {
		t.Error("JoinDeadline must stay cleared across reconnects")
	}
}

// ---- Watchdog auto-cancel on JoinDeadline --------------------------------

// TestWatchdogAutoCancelsWhenJoinDeadlinePasses creates a play game whose
// JoinDeadline is already in the past, then waits for the 100ms-tick
// watchdog goroutine to abandon it.
func TestWatchdogAutoCancelsWhenJoinDeadlinePasses(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())

	past := time.Now().Add(-time.Second)
	pm := newTestPlayMeta(&past)
	mode := game.GameMode{Name: "Blitz 5+2", TimeNs: 5 * 60 * 1_000_000_000, Increment: 2 * 1_000_000_000, Variant: "Standard", Category: "blitz", Timed: true}

	g := game.NewPlayGame(&mode, pm)
	if !g.IsOngoing() {
		t.Fatal("game should start as ongoing")
	}

	if !waitUntil(func() bool { return !g.IsOngoing() }, 2*time.Second) {
		t.Fatal("watchdog should have abandoned the game after JoinDeadline passed")
	}
	if g.State != game.GameAbandoned {
		t.Errorf("state = %v, want GameAbandoned", g.State)
	}
	if g.Winner != engine.NoColor {
		t.Errorf("winner = %v, want NoColor (no-one joined, so no win)", g.Winner)
	}
}

// TestWatchdogDoesNotCancelBeforeJoinDeadline ensures the watchdog leaves the
// game alone while the deadline is in the future and only the creator is
// connected. We pick a far-future deadline and sleep a few watchdog ticks.
func TestWatchdogDoesNotCancelBeforeJoinDeadline(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())

	future := time.Now().Add(30 * time.Second)
	pm := newTestPlayMeta(&future)
	mode := game.GameMode{Name: "Blitz 5+2", TimeNs: 5 * 60 * 1_000_000_000, Increment: 2 * 1_000_000_000, Variant: "Standard", Category: "blitz", Timed: true}

	g := game.NewPlayGame(&mode, pm)
	// Give the watchdog several poll cycles (100ms each).
	time.Sleep(400 * time.Millisecond)
	if !g.IsOngoing() {
		t.Fatalf("watchdog must not abandon a game whose JoinDeadline is in the future; got state=%v", g.State)
	}
}

// TestWatchdogDoesNotCancelOnceBothConnected ensures that even if the
// deadline somehow lingered (defensive), once both players have first
// connected the watchdog stops treating it as a join-timeout (the
// BothPlayersConnectedOnce branch takes over).
func TestWatchdogDoesNotCancelAfterBothConnected(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())

	past := time.Now().Add(-time.Second)
	pm := newTestPlayMeta(&past)
	mode := game.GameMode{Name: "Blitz 5+2", TimeNs: 5 * 60 * 1_000_000_000, Increment: 2 * 1_000_000_000, Variant: "Standard", Category: "blitz", Timed: true}

	// Mark both players as connected BEFORE starting the watchdog can see
	// the deadline. RecordSSEConnect for both also clears JoinDeadline.
	pm.RecordSSEConnect("white")
	pm.RecordSSEConnect("black")

	g := game.NewPlayGame(&mode, pm)
	time.Sleep(300 * time.Millisecond)
	if !g.IsOngoing() {
		t.Errorf("watchdog must not abandon once both players have connected; got state=%v", g.State)
	}
}
