package game

import (
	"encoding/json"
	"fmt"
	"sync"
)

// GameHub is a fan-out broadcast hub for SSE subscribers.
// Each connected browser tab gets its own buffered channel.
type GameHub struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func NewGameHub() *GameHub {
	return &GameHub{
		subs: make(map[chan []byte]struct{}),
	}
}

// Subscribe returns a buffered channel that receives broadcast messages.
func (h *GameHub) Subscribe() chan []byte {
	ch := make(chan []byte, 32)
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

// Broadcast sends msg to every subscriber, dropping the message for any
// subscriber whose buffer is full (non-blocking send).
func (h *GameHub) Broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

// Len returns the number of active subscribers.
func (h *GameHub) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// BroadcastEvent marshals v as JSON and broadcasts it to all subscribers.
func (h *GameHub) BroadcastEvent(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	raw := fmt.Appendf(nil, "data: %s\n\n", b)
	h.Broadcast(raw)
}
