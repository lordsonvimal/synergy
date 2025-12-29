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

var gameModeMap map[string]GameMode

func init() {
	gameModeMap = make(map[string]GameMode)

	modes := []GameMode{
		{Name: "Standard 5+2", TimeNs: 5 * 60 * 1_000_000_000, Increment: 2 * 1_000_000_000, Variant: "Standard"},
		{Name: "Blitz 3+0", TimeNs: 3 * 60 * 1_000_000_000, Increment: 0, Variant: "Standard"},
		{Name: "Rapid 10+5", TimeNs: 10 * 60 * 1_000_000_000, Increment: 5 * 1_000_000_000, Variant: "Standard"},
	}

	for _, gm := range modes {
		key := strings.ToLower(strings.TrimSpace(gm.Name))
		gameModeMap[key] = gm
	}
}

// FindGameModeByName efficiently returns the GameMode based on user input
func FindGameModeByName(selection string) (GameMode, error) {
	key := strings.ToLower(strings.TrimSpace(selection))
	if gm, ok := gameModeMap[key]; ok {
		return gm, nil
	}
	return GameMode{}, errors.New("invalid game mode selection")
}
