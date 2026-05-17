package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

// Datastar SSE constants. Defined here (rather than imported from datastar-go)
// to avoid pulling that dependency into the game package.
const (
	datastarPatchSignalsEvent = "datastar-patch-signals"
	datastarSignalsLine       = "signals "
)

// subscriberBufSize is the per-subscriber SSE channel buffer. Sized so that
// even a brief reader stall (e.g. browser GC) doesn't immediately trigger
// the coalesce path. Empirically 32 was too small under bursty clock-tick +
// move traffic on slow networks.
const subscriberBufSize = 128

// GameHub is a fan-out broadcast hub for SSE subscribers.
// Each connected browser tab gets its own buffered channel.
type GameHub struct {
	mu       sync.Mutex
	subs     map[chan []byte]struct{}
	coalesce atomic.Uint64 // count of messages dropped to make room for newer ones
}

func NewGameHub() *GameHub {
	return &GameHub{
		subs: make(map[chan []byte]struct{}),
	}
}

// Subscribe returns a buffered channel that receives broadcast messages.
func (h *GameHub) Subscribe() chan []byte {
	ch := make(chan []byte, subscriberBufSize)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel and closes it.
func (h *GameHub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
	close(ch)
}

// Broadcast sends msg to every subscriber. If a subscriber's buffer is full
// (slow reader), the oldest queued frame is dropped to make room for the
// newest — because every frame this hub carries is a full board+signals
// snapshot, the newer one strictly supersedes the older one. This replaces
// the previous silent-drop behavior, which could leave a slow client with a
// stale view forever if the queue filled at exactly the wrong moment (the
// "moves applied then suddenly reverted" symptom on low-connectivity
// devices).
func (h *GameHub) Broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		h.coalesce.Add(sendCoalesce(ch, msg))
	}
}

// CoalesceCount returns the total number of frames that have been dropped
// across all subscribers to make room for newer ones. Exposed for tests and
// future telemetry — a non-zero value indicates at least one subscriber is
// reading slower than the broadcast rate.
func (h *GameHub) CoalesceCount() uint64 {
	return h.coalesce.Load()
}

// sendCoalesce pushes msg into ch, dropping the oldest queued message if ch
// is full. Returns the number of frames dropped (0 in the common case).
func sendCoalesce(ch chan []byte, msg []byte) uint64 {
	var dropped uint64
	for {
		select {
		case ch <- msg:
			return dropped
		default:
			// Buffer full — drop the oldest queued frame and retry. The
			// inner select is non-blocking too: if the reader drained one
			// between our two selects, we just retry the send.
			select {
			case <-ch:
				dropped++
			default:
			}
		}
	}
}

// Len returns the number of active subscribers.
func (h *GameHub) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// BroadcastSignals broadcasts a datastar signal patch as a SSE frame to all
// subscribers. Replaces the older plain-JSON BroadcastEvent so that there is
// a single SSE protocol on the wire (datastar's), eliminating the need for a
// separate native EventSource on the client.
func (h *GameHub) BroadcastSignals(signals map[string]any) {
	b, err := json.Marshal(signals)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "event: %s\n", datastarPatchSignalsEvent)
	fmt.Fprintf(&buf, "data: %s%s\n\n", datastarSignalsLine, b)
	h.Broadcast(buf.Bytes())
}
