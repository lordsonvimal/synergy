package helpers

import (
	"fmt"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
)

func FormatTime(ns int64) string {
	mins := ns / 1_000_000_000 / 60
	return fmt.Sprintf("%dm", mins)
}

func FormatInc(ns int64) string {
	secs := ns / 1_000_000_000
	return fmt.Sprintf("%ds", secs)
}

func SquareName(sq uint8) string {
	return fmt.Sprintf("%c%d", 'a'+sq%8, sq/8+1)
}

func PieceName(piece engine.Piece) string {
	switch piece {
	case engine.Pawn:
		return "Pawn"
	case engine.Knight:
		return "Knight"
	case engine.Bishop:
		return "Bishop"
	case engine.Rook:
		return "Rook"
	case engine.Queen:
		return "Queen"
	case engine.King:
		return "King"
	default:
		return ""
	}
}

func ColorName(color engine.Color) string {
	if color == engine.White {
		return "White"
	}
	return "Black"
}

func SquareLabel(sq uint8, color engine.Color, piece engine.Piece, occupied bool) string {
	name := SquareName(sq)
	if !occupied {
		return name
	}
	return fmt.Sprintf("%s, %s %s", name, ColorName(color), PieceName(piece))
}
