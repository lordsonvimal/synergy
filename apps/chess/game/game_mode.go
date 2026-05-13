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
	Category  string // "blitz", "rapid", "classical", "unlimited"
	Timed     bool
}

type GameModeGroup struct {
	Category string
	Label    string
	Modes    []GameMode
}

var (
	gameModeMap map[string]GameMode
	gameModes   []GameMode
)

func init() {
	gameModeMap = make(map[string]GameMode)

	gameModes = []GameMode{
		// Blitz
		{Name: "Blitz 3+0", TimeNs: 3 * 60 * ns, Increment: 0, Variant: "Standard", Category: "blitz", Timed: true},
		{Name: "Blitz 3+1", TimeNs: 3 * 60 * ns, Increment: 1 * ns, Variant: "Standard", Category: "blitz", Timed: true},
		{Name: "Blitz 5+0", TimeNs: 5 * 60 * ns, Increment: 0, Variant: "Standard", Category: "blitz", Timed: true},
		{Name: "Blitz 5+2", TimeNs: 5 * 60 * ns, Increment: 2 * ns, Variant: "Standard", Category: "blitz", Timed: true},
		// Rapid
		{Name: "Rapid 10+0", TimeNs: 10 * 60 * ns, Increment: 0, Variant: "Standard", Category: "rapid", Timed: true},
		{Name: "Rapid 10+5", TimeNs: 10 * 60 * ns, Increment: 5 * ns, Variant: "Standard", Category: "rapid", Timed: true},
		// Classical
		{Name: "Classical 30+0", TimeNs: 30 * 60 * ns, Increment: 0, Variant: "Standard", Category: "classical", Timed: true},
		{Name: "Classical 30+5", TimeNs: 30 * 60 * ns, Increment: 5 * ns, Variant: "Standard", Category: "classical", Timed: true},
		{Name: "Classical 60+0", TimeNs: 60 * 60 * ns, Increment: 0, Variant: "Standard", Category: "classical", Timed: true},
		{Name: "Classical 60+10", TimeNs: 60 * 60 * ns, Increment: 10 * ns, Variant: "Standard", Category: "classical", Timed: true},
	}

	for _, gm := range gameModes {
		gameModeMap[normalizeModeKey(gm.Name)] = gm
	}

	u := UnlimitedMode()
	gameModeMap[normalizeModeKey(u.Name)] = u
}

const ns = int64(1_000_000_000)

// UnlimitedMode returns the no-clock mode used for solo and computer play.
func UnlimitedMode() GameMode {
	return GameMode{Name: "Unlimited", TimeNs: 0, Increment: 0, Variant: "Standard", Category: "unlimited", Timed: false}
}

// ListOnlineModeGroups returns the online modes grouped by category in display order.
func ListOnlineModeGroups() []GameModeGroup {
	groups := []GameModeGroup{
		{Category: "blitz", Label: "Blitz"},
		{Category: "rapid", Label: "Rapid"},
		{Category: "classical", Label: "Classical"},
	}
	for _, gm := range gameModes {
		for i := range groups {
			if groups[i].Category == gm.Category {
				groups[i].Modes = append(groups[i].Modes, gm)
				break
			}
		}
	}
	return groups
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
