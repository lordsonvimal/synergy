package game

import (
	"time"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
)

type Clock struct {
	RemainingNs int64
	LastStartNs int64
	Running     bool
}

type GameClock struct {
	White Clock
	Black Clock
	IncNs int64
	Turn  int
}

// NewClock returns a GameClock with the given initial time and increment (both in nanoseconds)
func NewClock(initialTimeNs int64, incNs int64) GameClock {
	return GameClock{
		White: Clock{
			RemainingNs: initialTimeNs,
			LastStartNs: 0,
			Running:     false,
		},
		Black: Clock{
			RemainingNs: initialTimeNs,
			LastStartNs: 0,
			Running:     false,
		},
		IncNs: incNs,
		Turn:  0,
	}
}

func monoNow() int64 {
	return time.Now().UnixNano()
}

func (gc *GameClock) Start(color engine.Color) {
	now := monoNow()
	c := gc.clock(color)
	c.LastStartNs = now
	c.Running = true
}

func (gc *GameClock) Stop(color engine.Color, lagCompNs int64) {
	now := monoNow()
	c := gc.clock(color)

	// If the clock was not running (e.g. white's first move before the play
	// clock has started ticking), do not consume time and do not award the
	// increment — the player effectively used no time.
	if !c.Running {
		gc.Turn++
		return
	}

	elapsed := now - c.LastStartNs - lagCompNs
	if elapsed < 0 {
		elapsed = 0
	}
	c.RemainingNs -= elapsed
	if c.RemainingNs < 0 {
		c.RemainingNs = 0
	}

	c.RemainingNs += gc.IncNs
	c.Running = false
	gc.Turn++
}

func (gc *GameClock) clock(color engine.Color) *Clock {
	if color == 0 {
		return &gc.White
	}
	return &gc.Black
}

// RemainingAt returns the remaining nanoseconds for the given side at the
// given unix nanosecond timestamp, accounting for elapsed time if running.
func (gc *GameClock) RemainingAt(color engine.Color, nowNs int64) int64 {
	c := gc.clock(color)
	if !c.Running {
		return c.RemainingNs
	}
	elapsed := nowNs - c.LastStartNs
	if elapsed < 0 {
		elapsed = 0
	}
	rem := c.RemainingNs - elapsed
	if rem < 0 {
		return 0
	}
	return rem
}
