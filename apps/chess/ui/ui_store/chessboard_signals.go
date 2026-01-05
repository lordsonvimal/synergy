package ui_store

import (
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
)

type ChessBoardSignals struct {
	SelectedSquare uint8          `json:"selectedSquare"`
	SideToMove     engine.Color   `json:"sideToMove"`
	PossibleMoves  []int          `json:"possibleMoves"`
	Promotion      bool           `json:"promotion"`
	PromotedSquare uint8          `json:"promotedSquare"`
	PromotionPiece engine.Piece   `json:"promotionPiece"`
	GameState      game.GameState `json:"gameState"`
	GameStateText  string         `json:"gameStateText"` // <- new
	IsCheck        bool           `json:"isCheck"`
	Winner         engine.Color   `json:"winner"` // nil if game ongoing / draw
}

func NewChessBoardSignals() *ChessBoardSignals {
	return &ChessBoardSignals{
		SelectedSquare: engine.NoSquare,
		SideToMove:     engine.White,
		PossibleMoves:  []int{},
		Promotion:      false,
		PromotedSquare: 255,
		PromotionPiece: engine.NoPiece,
		GameState:      game.GameOngoing,
		GameStateText:  "Ongoing",
		IsCheck:        false,
		Winner:         engine.NoColor,
	}
}

func (s *ChessBoardSignals) ClearSelection() {
	s.SelectedSquare = 255
	s.PossibleMoves = []int{}
}

func (s *ChessBoardSignals) ClearPossibleMoves() {
	s.PossibleMoves = []int{}
}

func (s *ChessBoardSignals) EnablePromotion(promotedSquare uint8) {
	s.Promotion = true
	s.PossibleMoves = []int{}
	s.PromotedSquare = promotedSquare
}

func (s *ChessBoardSignals) ClearPromotion() {
	s.Promotion = false
	s.PromotedSquare = 255
	s.PromotionPiece = engine.NoPiece
}

func (s *ChessBoardSignals) UpdateFromGame(g *game.Game) {
	s.SideToMove = g.Board.SideToMove

	// Update game state
	s.IsCheck = g.IsCheck()
	s.GameState = g.State
	if g.State != game.GameOngoing && g.Winner != engine.NoColor {
		c := g.Winner
		s.Winner = c
	} else {
		s.Winner = engine.NoColor
	}

	// Set human-readable GameState text
	switch g.State {
	case game.GameOngoing:
		s.GameStateText = "Ongoing"
	case game.GameCheckmate:
		s.GameStateText = "Checkmate"
	case game.GameResigned:
		s.GameStateText = "Resigned"
	case game.GameClockFlagged:
		s.GameStateText = "Clock flagged"
	case game.GameDrawStalemate:
		s.GameStateText = "Stalemate"
	case game.GameDrawFiftyMove:
		s.GameStateText = "Fifty-move rule"
	case game.GameDrawAgreement:
		s.GameStateText = "Draw by agreement"
	case game.GameDrawThreefoldRepetition:
		s.GameStateText = "Threefold repetition"
	case game.GameDrawInsufficientMaterial:
		s.GameStateText = "Insufficient material"
	case game.GameAbandoned:
		s.GameStateText = "Abandoned"
	case game.GameDisconnected:
		s.GameStateText = "Disconnected"
	case game.GameInvalid:
		s.GameStateText = "Invalid"
	default:
		s.GameStateText = "Unknown"
	}

	// Update selection
	// Update selection and possible moves
	if g.Selection != nil {
		s.SelectedSquare = g.Selection.FromSquare
		s.PossibleMoves = make([]int, len(g.Selection.Targets))
		for i, t := range g.Selection.Targets {
			s.PossibleMoves[i] = int(t)
		}
	} else {
		s.SelectedSquare = engine.NoSquare
		s.PossibleMoves = []int{}
	}
}
