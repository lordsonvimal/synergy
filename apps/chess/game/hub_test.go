package game_test

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/lordsonvimal/synergy/apps/chess/game"
)

func TestHub_BroadcastDeliversToAllSubscribers(t *testing.T) {
	h := game.NewGameHub()
	a := h.Subscribe()
	defer h.Unsubscribe(a)
	b := h.Subscribe()
	defer h.Unsubscribe(b)

	h.Broadcast([]byte("msg"))

	if got := <-a; string(got) != "msg" {
		t.Fatalf("subscriber a got %q", got)
	}
	if got := <-b; string(got) != "msg" {
		t.Fatalf("subscriber b got %q", got)
	}
}

func TestHub_FullBufferCoalescesToLatest(t *testing.T) {
	h := game.NewGameHub()
	slow := h.Subscribe()
	defer h.Unsubscribe(slow)

	// Flood the hub with strictly more frames than the per-subscriber buffer
	// can hold. With the old non-blocking-drop policy the slow subscriber
	// would have seen the FIRST N frames and missed everything after; with
	// coalesce-to-latest it should always end up holding the most recent
	// frame as the head of its drain.
	const total = 500
	for i := 0; i < total; i++ {
		h.Broadcast([]byte("frame-" + strconv.Itoa(i)))
	}

	// Drain everything currently buffered and check the final message is the
	// last broadcast. The intermediate frames may have been coalesced away —
	// that's the point.
	var last string
	for {
		select {
		case msg, ok := <-slow:
			if !ok {
				t.Fatal("channel closed unexpectedly")
			}
			last = string(msg)
		default:
			if last != "frame-"+strconv.Itoa(total-1) {
				t.Fatalf("slow subscriber's final drained frame = %q, want frame-%d", last, total-1)
			}
			if h.CoalesceCount() == 0 {
				t.Fatal("expected non-zero coalesce count after flooding past buffer size")
			}
			return
		}
	}
}

func TestHub_ConcurrentBroadcastSafe(t *testing.T) {
	h := game.NewGameHub()
	sub := h.Subscribe()
	defer h.Unsubscribe(sub)

	// Drain in background so the channel doesn't fill and stop the race
	// detector from observing real concurrent writes to the channel.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case _, ok := <-sub:
				if !ok {
					return
				}
			case <-time.After(50 * time.Millisecond):
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				h.Broadcast([]byte("x"))
			}
		}()
	}
	wg.Wait()
	<-done
}
