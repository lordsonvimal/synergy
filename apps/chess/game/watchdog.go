package game

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	watchdogPollInterval = 100 * time.Millisecond
	tickBroadcastInterval = time.Second
)

// startWatchdog launches a goroutine that:
//  - polls the active clock every 100ms for flag fall
//  - broadcasts a ClockTickEvent to all SSE subscribers once per second
//  - broadcasts a GameOverEvent and stops when the game ends
//
// It exits when the game is over or ctx is cancelled via the game's stopWatchdog channel.
func (g *Game) startWatchdog() {
	go func() {
		ticker := time.NewTicker(watchdogPollInterval)
		defer ticker.Stop()

		var lastTick time.Time

		for {
			select {
			case <-g.stopCh:
				return
			case now := <-ticker.C:
				g.mu.Lock()

				if g.State != GameOngoing {
					g.mu.Unlock()
					return
				}

				// Compute live remaining for both sides.
				activeSide := g.Board.SideToMove
				activeClock := g.Clock.clock(activeSide)
				nowNs := now.UnixNano()
				whiteRem := g.Clock.RemainingAt(0, nowNs)
				blackRem := g.Clock.RemainingAt(1, nowNs)

				// Flag fall check — authoritative.
				if activeClock.Running && (whiteRem <= 0 || blackRem <= 0) {
					g.State = GameClockFlagged
					if whiteRem <= 0 {
						whiteRem = 0
						g.Winner = 1 // Black wins
					} else {
						blackRem = 0
						g.Winner = 0 // White wins
					}
					activeClock.Running = false
					g.signalGameOver()
					g.mu.Unlock()
					return
				}

				// 1 Hz clock tick broadcast.
				if now.Sub(lastTick) >= tickBroadcastInterval {
					lastTick = now
					evt := ClockTickEvent{
						Type:             "clock_tick",
						WhiteRemainingNs: whiteRem,
						BlackRemainingNs: blackRem,
						WhiteRunning:     g.Clock.White.Running,
						BlackRunning:     g.Clock.Black.Running,
						ServerTsNs:       nowNs,
					}
					g.mu.Unlock()
					g.broadcastJSON(evt)
				} else {
					g.mu.Unlock()
				}
			}
		}
	}()
}

// broadcastJSON marshals v and broadcasts it as an unnamed SSE data event
// ("data: <json>\n\n") so GameEventsHandler can write the bytes directly.
func (g *Game) broadcastJSON(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	raw := fmt.Appendf(nil, "data: %s\n\n", b)
	g.Hub.Broadcast(raw)
}
