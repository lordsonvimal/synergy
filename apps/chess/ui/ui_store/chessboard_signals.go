package ui_store

import (
	"encoding/json"

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
	GameStateText  string         `json:"gameStateText"`
	IsCheck        bool           `json:"isCheck"`
	Winner         engine.Color   `json:"winner"`
	Timed          bool           `json:"timed"`
	ClientTsNs     int64          `json:"clientTsNs"`    // unix ns set by client on move click; used for lag compensation
	SelectionSeq   int64          `json:"selectionSeq"`  // client-incremented per click; read on the request, echoed as serverSelectionSeq in responses
}

// MarshalJSON renames the inbound selectionSeq field to serverSelectionSeq on
// the way out so server responses never clobber the client's monotonic
// selectionSeq counter. The client's $selectionSeq is the source of truth for
// "latest intent"; $serverSelectionSeq is what the server has acknowledged.
func (s *ChessBoardSignals) MarshalJSON() ([]byte, error) {
	type alias ChessBoardSignals
	b, err := json.Marshal((*alias)(s))
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	m["serverSelectionSeq"] = s.SelectionSeq
	delete(m, "selectionSeq")
	return json.Marshal(m)
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
	s.Timed = g.Timed
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
