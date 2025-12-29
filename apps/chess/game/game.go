package game

import "github.com/lordsonvimal/synergy/apps/chess/engine"

type Game struct {
	ID    string
	Board engine.Board
	Clock GameClock
	WAL   WAL
	Seq   uint64
}

// --------------------------
// Check if current side's king is in check
// --------------------------
func (g *Game) IsCheck() bool {
	return g.Board.IsKingInCheck(g.Board.SideToMove)
}

// --------------------------
// Check if current side is checkmated
// --------------------------
func (g *Game) IsCheckmate() bool {
	return g.Board.IsKingInCheck(g.Board.SideToMove) && !g.Board.HasLegalMoves(g.Board.SideToMove)
}

// --------------------------
// Check if current side is stalemated
// --------------------------
func (g *Game) IsStalemate() bool {
	return !g.Board.IsKingInCheck(g.Board.SideToMove) && !g.Board.HasLegalMoves(g.Board.SideToMove)
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
