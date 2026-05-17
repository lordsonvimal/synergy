package game_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lordsonvimal/synergy/apps/chess/game"
)

// TestMoveBatch_FlushNowWaitsForInFlightWrite verifies that FlushNow blocks
// until any concurrent Append-spawned flush has actually persisted, not just
// drained the buffer. This is the invariant the move-POST path relies on
// for durability: by the time we 200, the row is on disk.
func TestMoveBatch_FlushNowWaitsForInFlightWrite(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var written atomic.Int32

	flushFn := func(_ context.Context, moves []game.PendingMove, events []game.PendingEvent) error {
		close(started)
		// Hold up the "write" so we can call FlushNow while it's pending.
		<-release
		written.Add(int32(len(moves)))
		return nil
	}

	b := game.NewMoveBatch("sid", flushFn, nil, nil)
	defer b.FlushAndStop(context.Background(), "", "")

	// Append spawns a goroutine that calls flush(). It will lock flushMu
	// and block inside flushFn until we close `release`.
	b.Append(game.PendingMove{Seq: 1}, game.PendingEvent{})
	<-started // the goroutine is now inside flushFn, holding flushMu

	// Call FlushNow concurrently. Its inner flush() drain will find an empty
	// buffer (the Append goroutine already drained), then it must block on
	// flushMu until the in-flight write completes.
	done := make(chan struct{})
	go func() {
		b.FlushNow(context.Background())
		close(done)
	}()

	// Verify FlushNow has not returned yet — proves it's waiting on flushMu.
	select {
	case <-done:
		t.Fatal("FlushNow returned before the in-flight write completed")
	case <-time.After(50 * time.Millisecond):
		// expected: still blocked
	}

	// Release the in-flight write. FlushNow should now return promptly.
	close(release)
	select {
	case <-done:
		// expected
	case <-time.After(time.Second):
		t.Fatal("FlushNow did not return after the in-flight write completed")
	}

	if written.Load() != 1 {
		t.Fatalf("expected exactly one move written, got %d", written.Load())
	}
}

func TestMoveBatch_FlushNowFlushesIfNotInFlight(t *testing.T) {
	var written atomic.Int32
	flushFn := func(_ context.Context, moves []game.PendingMove, _ []game.PendingEvent) error {
		written.Add(int32(len(moves)))
		return nil
	}
	b := game.NewMoveBatch("sid", flushFn, nil, nil)
	defer b.FlushAndStop(context.Background(), "", "")

	// Drop a move directly into the buffer without triggering the async path.
	// We need the buffer non-empty when FlushNow runs — Append's goroutine
	// may race us, but multiple flushes are idempotent.
	b.Append(game.PendingMove{Seq: 1}, game.PendingEvent{})

	// Give Append's goroutine time to either fully flush or get past the
	// buffer-drain (the synchronous FlushNow below covers both cases).
	b.FlushNow(context.Background())

	if got := written.Load(); got < 1 {
		t.Fatalf("expected at least one write, got %d", got)
	}
}
