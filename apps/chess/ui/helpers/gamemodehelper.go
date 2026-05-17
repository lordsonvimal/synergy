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

// FormatTimeControl returns the compact "3+0" display label for a time control.
func FormatTimeControl(timeNs, incNs int64) string {
	mins := timeNs / 1_000_000_000 / 60
	secs := incNs / 1_000_000_000
	return fmt.Sprintf("%d+%d", mins, secs)
}

// FormatTimeControlSubtitle returns a descriptive subtitle for a time control,
// e.g. "5 min · no inc" or "5 min · +2s / move".
func FormatTimeControlSubtitle(timeNs, incNs int64) string {
	mins := timeNs / 1_000_000_000 / 60
	secs := incNs / 1_000_000_000
	if secs == 0 {
		return fmt.Sprintf("%d min · no inc", mins)
	}
	return fmt.Sprintf("%d min · +%ds / move", mins, secs)
}

// CategoryTagline returns a short description for a game-mode category,
// used as a section subtitle on the game-mode selection page.
func CategoryTagline(category string) string {
	switch category {
	case "blitz":
		return "Fast-paced — under 5 minutes per side"
	case "rapid":
		return "Standard pace — around 10 minutes per side"
	case "classical":
		return "Long-form — 30 minutes or more per side"
	default:
		return ""
	}
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
