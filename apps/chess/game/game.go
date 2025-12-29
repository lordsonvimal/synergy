package game

import (
	"sync"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
)

type Game struct {
	ID    string
	Board engine.Board
	Clock GameClock
	WAL   WAL
	Seq   uint64

	mu             sync.RWMutex
	legalMoveCache map[engine.Color]bool // cache per side
}

// --------------------------
// Check if current side's king is in check
// --------------------------
func (g *Game) IsCheck() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Board.IsKingInCheck(g.Board.SideToMove)
}

// --------------------------
// Check if current side is checkmated
// --------------------------
func (g *Game) IsCheckmate() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	color := g.Board.SideToMove

	// If king is not in check, cannot be checkmate
	if !g.Board.IsKingInCheck(color) {
		return false
	}

	// Check legal moves lazily
	return !g.hasLegalMoves(color)
}

// --------------------------
// Check if current side is stalemated
// --------------------------
func (g *Game) IsStalemate() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	color := g.Board.SideToMove

	// If king is in check, cannot be stalemate
	if g.Board.IsKingInCheck(color) {
		return false
	}

	// Check legal moves lazily
	return !g.hasLegalMoves(color)
}

// --------------------------
// Apply move
// --------------------------
func (g *Game) ApplyMove(m engine.Move, lagCompNs int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	color := g.Board.SideToMove
	g.Clock.Stop(color, lagCompNs)

	if !g.Board.MakeMove(m) {
		return false
	}

	// Reset legal move cache since board changed
	g.legalMoveCache = nil

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

// --------------------------
// Helper: cached HasLegalMoves
// --------------------------
func (g *Game) hasLegalMoves(color engine.Color) bool {
	if g.legalMoveCache == nil {
		g.legalMoveCache = make(map[engine.Color]bool)
	}
	if val, ok := g.legalMoveCache[color]; ok {
		return val
	}
	val := g.Board.HasLegalMoves(color)
	g.legalMoveCache[color] = val
	return val
}
