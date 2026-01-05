package ui_store

import "github.com/lordsonvimal/synergy/apps/chess/engine"

type ChessBoardSignals struct {
	SelectedSquare uint8        `json:"selectedSquare"`
	PossibleMoves  []uint8      `json:"possibleMoves"`
	Promotion      bool         `json:"promotion"`
	PromotedSquare uint8        `json:"promotedSquare"`
	PromotionPiece engine.Piece `json:"promotionPiece"`
}

func NewChessBoardSignals() *ChessBoardSignals {
	return &ChessBoardSignals{
		SelectedSquare: 255,
		PossibleMoves:  []uint8{},
		Promotion:      false,
		PromotedSquare: 255,
		PromotionPiece: engine.NoPiece,
	}
}

func (s *ChessBoardSignals) ClearSelection() {
	s.SelectedSquare = 255
	s.PossibleMoves = nil
}

func (s *ChessBoardSignals) ClearPossibleMoves() {
	s.PossibleMoves = nil
}

func (s *ChessBoardSignals) ClearPromotion() {
	s.Promotion = false
	s.PromotedSquare = 255
	s.PromotionPiece = engine.NoPiece
}
