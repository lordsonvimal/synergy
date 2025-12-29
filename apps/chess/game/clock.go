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
	elapsed := now - c.LastStartNs - lagCompNs
	if elapsed < 0 {
		elapsed = 0
	}
	c.RemainingNs -= elapsed
	c.RemainingNs += gc.IncNs
	c.Running = false
}

func (gc *GameClock) clock(color engine.Color) *Clock {
	if color == 0 {
		return &gc.White
	}
	return &gc.Black
}
