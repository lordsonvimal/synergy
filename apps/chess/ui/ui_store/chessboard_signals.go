package ui_store

import (
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
)

// (selectionSeq round-trip / MarshalJSON tricks removed: selection is now
// fully client-side via chessops, so there's no per-mover ack to plumb.)

type ChessBoardSignals struct {
	SelectedSquare uint8          `json:"selectedSquare"`
	SideToMove     engine.Color   `json:"sideToMove"`
	PossibleMoves  []int          `json:"possibleMoves"`
	Promotion      bool           `json:"promotion"`
	PromotedSquare uint8          `json:"promotedSquare"`
	PromotionPiece engine.Piece   `json:"promotionPiece"`
	GameState      game.GameState `json:"gameState"`
	GameStateText  string         `json:"gameStateText"`
	IsCheck        bool           `json:"isCheck"`
	Winner         engine.Color   `json:"winner"`
	Timed          bool           `json:"timed"`
	ClientTsNs     int64          `json:"clientTsNs"` // unix ns set by client on move; used for clock lag compensation
	// Fen is the authoritative position after every server-applied move.
	// board.js (chessops) re-anchors its local Chess instance whenever this
	// changes, so client-side legal-move generation stays in sync with the
	// server's engine. Empty before the first sync.
	Fen string `json:"fen"`
}

func NewChessBoardSignals() *ChessBoardSignals {
	return &ChessBoardSignals{
		SelectedSquare: engine.NoSquare, // 64 — must match board.js NO_SQUARE
		SideToMove:     engine.White,
		PossibleMoves:  []int{},
		Promotion:      false,
		PromotedSquare: engine.NoSquare,
		PromotionPiece: engine.NoPiece,
		GameState:      game.GameOngoing,
		GameStateText:  "Ongoing",
		IsCheck:        false,
		Winner:         engine.NoColor,
		Timed:          true,
	}
}

// ChessBoardSignalsFromGame creates signals pre-populated from the current game state.
// Use this when rendering an existing game (e.g. on page reload) so the initial
// DataStar signals match reality rather than defaulting to a fresh-game state.
func ChessBoardSignalsFromGame(g *game.Game) *ChessBoardSignals {
	s := NewChessBoardSignals()
	s.UpdateFromGame(g)
	return s
}

// RoleMatchesSide returns true when the given UI role ("white"/"black") is the
// side currently on move. Spectators and unknown roles return false.
func RoleMatchesSide(role string, side engine.Color) bool {
	switch role {
	case "white":
		return side == engine.White
	case "black":
		return side == engine.Black
	}
	return false
}

func (s *ChessBoardSignals) ClearSelection() {
	s.SelectedSquare = engine.NoSquare
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
	s.PromotedSquare = engine.NoSquare
	s.PromotionPiece = engine.NoPiece
}

func (s *ChessBoardSignals) UpdateFromGame(g *game.Game) {
	s.Timed = g.Timed
	s.SideToMove = g.Board.SideToMove
	s.Fen = g.Board.FEN()

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

	// Selection is now driven entirely by the client (chessops), but the
	// server still reflects whatever in-progress selection it knows about
	// — used on reconnect/spectator-load.
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
