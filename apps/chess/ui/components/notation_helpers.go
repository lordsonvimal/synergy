package components

import (
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
)

type notationRow struct {
	moveNumber       int
	white            string
	black            string
	whiteHalfMoveIdx int
	blackHalfMoveIdx int
}

// groupHistory pairs consecutive White/Black MoveRecords into display rows.
// Each row corresponds to one full-move number (e.g. "1. e4 e5").
// whiteHalfMoveIdx and blackHalfMoveIdx are 0-based indices into g.History.
func groupHistory(history []game.MoveRecord) []notationRow {
	var rows []notationRow
	for i, rec := range history {
		if rec.Color == engine.White {
			rows = append(rows, notationRow{
				moveNumber:       rec.MoveNumber,
				white:            rec.SAN,
				whiteHalfMoveIdx: i,
				blackHalfMoveIdx: -1,
			})
		} else if len(rows) > 0 && rows[len(rows)-1].moveNumber == rec.MoveNumber {
			rows[len(rows)-1].black = rec.SAN
			rows[len(rows)-1].blackHalfMoveIdx = i
		}
	}
	return rows
}
