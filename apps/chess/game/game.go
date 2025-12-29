package game

import "github.com/lordsonvimal/synergy/apps/chess/engine"

type Game struct {
	ID    string
	Board engine.Board
	Clock GameClock
	WAL   WAL
	Seq   uint64
}

func (g *Game) ApplyMove(m engine.Move, lagCompNs int64) bool {
	color := g.Board.SideToMove

	g.Clock.Stop(color, lagCompNs)

	if !g.Board.MakeMove(m) { // MakeMove must return bool
		return false
	}

	g.Clock.Start(color ^ 1)

	g.Seq++
	g.WAL.Append(WALEvent{
		Seq:       g.Seq,
		MoveUCI:   m.ToUCI(),
		ServerNs:  monoNow(),
		LagCompNs: lagCompNs,
		WRem:      g.Clock.White.RemainingNs,
		BRem:      g.Clock.Black.RemainingNs,
	})

	return true
}
