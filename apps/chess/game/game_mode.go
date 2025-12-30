package game

import (
	"errors"
	"strings"
)

type GameMode struct {
	Name      string
	TimeNs    int64
	Increment int64
	Variant   string
}

var (
	gameModeMap map[string]GameMode
	gameModes   []GameMode
)

func init() {
	gameModeMap = make(map[string]GameMode)

	gameModes = []GameMode{
		{
			Name:      "Standard 5+2",
			TimeNs:    5 * 60 * 1_000_000_000,
			Increment: 2 * 1_000_000_000,
			Variant:   "Standard",
		},
		{
			Name:      "Blitz 3+0",
			TimeNs:    3 * 60 * 1_000_000_000,
			Increment: 0,
			Variant:   "Standard",
		},
		{
			Name:      "Rapid 10+5",
			TimeNs:    10 * 60 * 1_000_000_000,
			Increment: 5 * 1_000_000_000,
			Variant:   "Standard",
		},
	}

	for _, gm := range gameModes {
		key := normalizeModeKey(gm.Name)
		gameModeMap[key] = gm
	}
}

// --------------------------
// Public API
// --------------------------

// FindGameModeByName returns a mode in O(1)
func FindGameModeByName(selection string) (GameMode, error) {
	key := normalizeModeKey(selection)
	if gm, ok := gameModeMap[key]; ok {
		return gm, nil
	}
	return GameMode{}, errors.New("invalid game mode selection")
}

// ListGameModes returns all available modes (for UI)
func ListGameModes() []GameMode {
	// return a copy to prevent mutation
	out := make([]GameMode, len(gameModes))
	copy(out, gameModes)
	return out
}

// --------------------------
// Helpers
// --------------------------

func normalizeModeKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
