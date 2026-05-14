package game

import "github.com/lordsonvimal/synergy/apps/chess/engine"

// MoveRecord stores the SAN notation and board snapshot for one played move.
// FEN reflects the position AFTER the move; SAN is computed BEFORE.
type MoveRecord struct {
	SAN        string       `json:"san"`
	FEN        string       `json:"fen"`
	Color      engine.Color `json:"color"`      // 0 = White, 1 = Black
	MoveNumber int          `json:"moveNumber"` // 1-based full-move count
	WRemNs     int64        `json:"w_rem_ns"`   // white remaining ns after this move
	BRemNs     int64        `json:"b_rem_ns"`   // black remaining ns after this move
}
