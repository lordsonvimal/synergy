package game

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
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

				// Play-mode gate: skip clock until both players have connected once.
				if g.PlayMeta != nil {
					pm := g.PlayMeta
					pm.mu.Lock()

					if !pm.BothPlayersConnectedOnce {
						pm.mu.Unlock()
						g.mu.Unlock()
						continue
					}

					// First-move timeout: 30s after both players first connected.
					if pm.FirstMoveDeadline != nil &&
						now.After(*pm.FirstMoveDeadline) &&
						len(g.History) == 0 {
						pm.mu.Unlock()
						g.State = GameAbandoned
						g.signalGameOver()
						g.mu.Unlock()
						return
					}

					// Rematch proposal expiry: 30s.
					if pm.RematchProposedAt != nil &&
						now.Sub(*pm.RematchProposedAt) > 30*time.Second {
						pm.RematchProposedBy = engine.NoColor
						pm.RematchProposedAt = nil
						pm.mu.Unlock()
						go g.broadcastJSON(RematchExpiredEvent{Type: "rematch_expired"})
					} else {
						pm.mu.Unlock()
					}

					// Re-lock pm to check disconnect thresholds.
					pm.mu.Lock()
					whiteDisNs := pm.totalDisconnectedNsLocked(engine.White, now)
					blackDisNs := pm.totalDisconnectedNsLocked(engine.Black, now)
					claimFor := pm.ClaimVictoryFor

					// 60s: enable claim-victory button (once per side).
					if claimFor == engine.NoColor {
						if whiteDisNs >= 60*int64(time.Second) {
							pm.ClaimVictoryFor = engine.White
							pm.mu.Unlock()
							go g.broadcastJSON(ClaimVictoryEvent{Type: "claim_victory_available", DisconnectedFor: "white"})
						} else if blackDisNs >= 60*int64(time.Second) {
							pm.ClaimVictoryFor = engine.Black
							pm.mu.Unlock()
							go g.broadcastJSON(ClaimVictoryEvent{Type: "claim_victory_available", DisconnectedFor: "black"})
						} else {
							pm.mu.Unlock()
						}
					} else {
						pm.mu.Unlock()
					}

					// 120s: auto-abandon.
					if whiteDisNs >= 120*int64(time.Second) || blackDisNs >= 120*int64(time.Second) {
						var winner engine.Color
						if whiteDisNs >= 120*int64(time.Second) {
							winner = engine.Black
						} else {
							winner = engine.White
						}
						g.State = GameAbandoned
						g.Winner = winner
						g.signalGameOver()
						g.mu.Unlock()
						return
					}
				}

				// Compute live remaining for both sides.
				activeSide := g.Board.SideToMove
				activeClock := g.Clock.clock(activeSide)
				nowNs := now.UnixNano()
				whiteRem := g.Clock.RemainingAt(0, nowNs)
				blackRem := g.Clock.RemainingAt(1, nowNs)

				// Flag fall check — authoritative.
				if g.Timed && activeClock.Running && (whiteRem <= 0 || blackRem <= 0) {
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
