package game

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/logger"
)

type SelectionState struct {
	FromSquare uint8
	Targets    []uint8
}

type SelectionSnapshot struct {
	FromSquare int   `json:"selectedSquare"`
	Targets    []int `json:"possibleMoves"`
}

type GameState int

const (
	GameOngoing                  GameState = iota
	GameCheckmate                          // checkmate
	GameResigned                           // player resigned
	GameClockFlagged                       // player ran out of time
	GameDrawStalemate                      // stalemate
	GameDrawFiftyMove                      // fifty-move rule
	GameDrawAgreement                      // mutual agreement
	GameDrawThreefoldRepetition            // threefold repetition
	GameDrawInsufficientMaterial           // insufficient material
	GameAbandoned                          // player left the game
	GameDisconnected                       // network error causing game to stop
	GameInvalid                            // invalid state
)

type Game struct {
	ID        string
	Board     *engine.Board
	Clock     GameClock
	WAL       *WAL
	Selection *SelectionState
	Seq       uint64
	State     GameState
	Winner    engine.Color // valid after game over

	mu             sync.RWMutex
	legalMoveCache map[engine.Color]bool // cache per side
}

func NewGame(mode *GameMode) *Game {
	board := engine.NewBoard()

	id := uuid.New().String()

	// 2. 5 minutes per side with 2-second increment
	gc := NewClock(mode.TimeNs, mode.Increment)

	wal, err := NewWAL("game_" + id + ".wal")
	if err != nil {
		log.Fatal(err)
	}

	// 4. Create the Game struct
	return &Game{
		ID:     id,
		Board:  board,
		Clock:  gc,
		WAL:    wal,
		Seq:    0,
		State:  GameOngoing,
		Winner: engine.NoColor,
	}
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
	// g.mu.RLock()
	// defer g.mu.RUnlock()

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

	// 0. Prevent moves if game is already over
	if g.State != GameOngoing {
		return false
	}

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

	g.ClearSelection()  // After move, clear selection
	g.UpdateGameState() // Update game state after each move
	return true
}

// --------------------------
// Update game state after a move
// --------------------------
func (g *Game) UpdateGameState() {
	// g.mu.Lock()
	// defer g.mu.Unlock()

	// Skip if game already over
	if g.State != GameOngoing {
		return
	}

	color := g.Board.SideToMove ^ 1 // The player who just moved

	// 1. Check checkmate
	if g.IsCheckmate() {
		g.State = GameCheckmate
		g.Winner = color // the player who just moved wins
		return
	}

	// 2. Check stalemate
	if g.IsStalemate() {
		g.State = GameDrawStalemate
		g.Winner = engine.NoColor
		return
	}

	// 3. Fifty-move rule
	// if g.Board.HalfMoveClock >= 100 {
	// 	g.State = GameDrawFiftyMove
	// 	g.Winner = engine.NoColor
	// 	return
	// }

	// 4. Threefold repetition
	if g.Board.IsThreefoldRepetition() {
		g.State = GameDrawThreefoldRepetition
		g.Winner = engine.NoColor
		return
	}

	// 5. Insufficient material
	if g.Board.IsInsufficientMaterial() {
		g.State = GameDrawInsufficientMaterial
		g.Winner = engine.NoColor
		return
	}

	// 6. Clock flag (time out)
	// if g.Clock.White.RemainingNs <= 0 {
	// 	g.State = GameClockFlagged
	// 	g.Winner = engine.Black
	// 	return
	// }
	// if g.Clock.Black.RemainingNs <= 0 {
	// 	g.State = GameClockFlagged
	// 	g.Winner = engine.White
	// 	return
	// }

	// 7. If none of the above, game ongoing
	g.State = GameOngoing
	g.Winner = engine.NoColor
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

// HasSelection returns true if a square is currently selected for the next move.
func (g *Game) HasSelection() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Selection != nil
}

// GetSelectionFrom returns the currently selected square index.
// Note: You should check HasSelection() before calling this.
func (g *Game) GetSelectionFrom() uint8 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Selection == nil {
		return 255 // Return invalid square if no selection
	}
	return g.Selection.FromSquare
}

func (g *Game) SelectionSnapshot() SelectionSnapshot {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.Selection == nil {
		return SelectionSnapshot{
			FromSquare: 255, // No selection
			Targets:    []int{},
		}
	}

	moves := make([]int, len(g.Selection.Targets))
	for i, t := range g.Selection.Targets {
		moves[i] = int(t)
	}

	return SelectionSnapshot{
		FromSquare: int(g.Selection.FromSquare),
		Targets:    moves,
	}
}

// IsTarget checks if the provided square is a valid move target for the current selection.
func (g *Game) IsTarget(square uint8) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Selection == nil {
		return false
	}
	for _, t := range g.Selection.Targets {
		if t == square {
			return true
		}
	}
	return false
}

func (g *Game) IsPromotionMove(move engine.Move) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	color, piece, _ := g.Board.PieceAt(move.From)
	if piece != engine.Pawn {
		return false
	}

	if !g.Board.TryMove(move) {
		return false
	}

	rank := move.To / 8
	if color == engine.White {
		return rank == 7
	}

	return rank == 0
}

func (g *Game) ClearSelection() {
	g.Selection = nil
}

func (g *Game) SelectSquare(ctx context.Context, square uint8) {
	g.mu.Lock()

	color, _, ok := g.Board.PieceAt(square)
	// If no piece or piece is not ours, clear selection
	if !ok || color != g.Board.SideToMove {
		g.Selection = nil
		g.mu.Unlock()

		logger.Info(ctx).Msg("Invalid piece")
		return
	}

	// Generate moves and update selection
	// Assuming you implement GenerateMovesForSquare in your engine
	moves := g.Board.GenerateMovesForSquare(square)

	targets := make([]uint8, 0, len(moves))
	for _, m := range moves {
		targets = append(targets, m.To)
	}

	g.Selection = &SelectionState{
		FromSquare: square,
		Targets:    targets,
	}

	g.mu.Unlock()

	logger.Info(ctx).
		Int("target lengths", len(moves)).
		Msg("SelectSquare EXIT: selected")
}
