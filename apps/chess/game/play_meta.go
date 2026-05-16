package game

import (
	"sync"
	"time"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
)

// PlayMeta holds all state specific to the online two-player (/play/*) flow.
// It is nil for solo (/game/*) games.
type PlayMeta struct {
	mu sync.Mutex

	// Set at creation; immutable thereafter.
	SessionID          string
	WhiteParticipantID string
	BlackParticipantID string
	WhiteToken         string
	BlackToken         string
	OriginalMode       GameMode // preserved for rematch game creation

	// Role claiming — backed by DB `claimed` column; restored on server restart.
	WhiteClaimed bool
	BlackClaimed bool

	// Active SSE connection counts.
	whiteConns int
	blackConns int

	// Cumulative disconnect time per side.
	whiteTotalDisconnectedNs int64
	blackTotalDisconnectedNs int64
	whiteDisconnectAt        *time.Time // nil = currently online
	blackDisconnectAt        *time.Time

	// Clock-unlock gate: true once both players have had ≥1 SSE connection.
	BothPlayersConnectedOnce bool
	FirstMoveDeadline        *time.Time // 30s after both first connect

	// ClockArmedAfterLoad tracks whether we've started the side-to-move clock
	// since this game was loaded into memory. Reset to false on each fresh
	// load (e.g. after a server restart) so the clock is re-armed when both
	// players reconnect — and not penalised for the downtime.
	ClockArmedAfterLoad bool

	// ClaimVictoryFor is set when a player has been disconnected ≥60s.
	ClaimVictoryFor engine.Color

	// Rematch state.
	RematchProposedBy engine.Color
	RematchProposedAt *time.Time
}

// RecordSSEConnect notes a new SSE connection for the given role.
// Returns:
//   - bothFirstConnected: true the first time ever both white and black are
//     simultaneously online (drives the initial first-move countdown).
//   - shouldArmClock: true when both sides are online and the clock has not
//     yet been armed since this game was loaded into memory. Used after a
//     server restart to resume an in-progress game's clock without
//     penalising either player for the downtime.
func (m *PlayMeta) RecordSSEConnect(role string) (bothFirstConnected, shouldArmClock bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	switch role {
	case "white":
		m.whiteConns++
		if m.whiteDisconnectAt != nil {
			m.whiteTotalDisconnectedNs += now.Sub(*m.whiteDisconnectAt).Nanoseconds()
			m.whiteDisconnectAt = nil
		}
	case "black":
		m.blackConns++
		if m.blackDisconnectAt != nil {
			m.blackTotalDisconnectedNs += now.Sub(*m.blackDisconnectAt).Nanoseconds()
			m.blackDisconnectAt = nil
		}
	default:
		return false, false
	}

	bothOnline := m.whiteConns > 0 && m.blackConns > 0
	if !m.BothPlayersConnectedOnce && bothOnline {
		m.BothPlayersConnectedOnce = true
		bothFirstConnected = true
	}
	if bothOnline && !m.ClockArmedAfterLoad {
		m.ClockArmedAfterLoad = true
		shouldArmClock = true
	}
	return bothFirstConnected, shouldArmClock
}

// RecordSSEDisconnect notes a dropped SSE connection for the given role.
func (m *PlayMeta) RecordSSEDisconnect(role string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	switch role {
	case "white":
		if m.whiteConns > 0 {
			m.whiteConns--
		}
		if m.whiteConns == 0 && m.whiteDisconnectAt == nil {
			m.whiteDisconnectAt = &now
		}
	case "black":
		if m.blackConns > 0 {
			m.blackConns--
		}
		if m.blackConns == 0 && m.blackDisconnectAt == nil {
			m.blackDisconnectAt = &now
		}
	}
}

// OnlineStatus returns whether white and black each have ≥1 active SSE connection.
func (m *PlayMeta) OnlineStatus() (white, black bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.whiteConns > 0, m.blackConns > 0
}

// totalDisconnectedNsLocked returns cumulative disconnected nanoseconds for a side.
// Caller must hold m.mu.
func (m *PlayMeta) totalDisconnectedNsLocked(color engine.Color, now time.Time) int64 {
	if color == engine.White {
		total := m.whiteTotalDisconnectedNs
		if m.whiteDisconnectAt != nil {
			total += now.Sub(*m.whiteDisconnectAt).Nanoseconds()
		}
		return total
	}
	total := m.blackTotalDisconnectedNs
	if m.blackDisconnectAt != nil {
		total += now.Sub(*m.blackDisconnectAt).Nanoseconds()
	}
	return total
}

// TryClaim atomically checks and claims a token.
// Returns participantID and role on success; ok=false if token is unknown or already claimed.
func (m *PlayMeta) TryClaim(token string) (participantID, role string, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch token {
	case m.WhiteToken:
		if !m.WhiteClaimed {
			m.WhiteClaimed = true
			return m.WhiteParticipantID, "white", true
		}
	case m.BlackToken:
		if !m.BlackClaimed {
			m.BlackClaimed = true
			return m.BlackParticipantID, "black", true
		}
	}
	return "", "", false
}

// GetRematchProposer returns the color that has a pending rematch proposal,
// or engine.NoColor if none is pending.
func (m *PlayMeta) GetRematchProposer() engine.Color {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.RematchProposedBy
}

// GetClaimVictoryFor returns the side whose disconnect enables claim-victory.
func (m *PlayMeta) GetClaimVictoryFor() engine.Color {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ClaimVictoryFor
}

// GetFirstMoveDeadlineNs returns the first-move deadline as unix nanoseconds,
// or 0 if no deadline has been set.
func (m *PlayMeta) GetFirstMoveDeadlineNs() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FirstMoveDeadline == nil {
		return 0
	}
	return m.FirstMoveDeadline.UnixNano()
}

// IsStarted returns true after both players have had ≥1 SSE connection.
func (m *PlayMeta) IsStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.BothPlayersConnectedOnce
}

// TryProposeRematch atomically sets a rematch proposal if none exists.
func (m *PlayMeta) TryProposeRematch(proposerColor engine.Color) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.RematchProposedBy != engine.NoColor {
		return false
	}
	now := time.Now()
	m.RematchProposedBy = proposerColor
	m.RematchProposedAt = &now
	return true
}

// AcceptAndClearRematch atomically accepts and clears a rematch proposal.
// Returns the proposer's color and true on success; false if the accepter is the proposer
// or no proposal exists.
func (m *PlayMeta) AcceptAndClearRematch(accepterColor engine.Color) (engine.Color, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.RematchProposedBy == engine.NoColor || m.RematchProposedBy == accepterColor {
		return engine.NoColor, false
	}
	proposer := m.RematchProposedBy
	m.RematchProposedBy = engine.NoColor
	m.RematchProposedAt = nil
	return proposer, true
}

// ClearRematch clears any pending rematch proposal.
func (m *PlayMeta) ClearRematch() {
	m.mu.Lock()
	m.RematchProposedBy = engine.NoColor
	m.RematchProposedAt = nil
	m.mu.Unlock()
}

// ClearRematchIfPendingBy clears the proposal only if the current proposer matches color.
// Returns true if cleared; false if no proposal or proposer differs (already accepted/declined).
func (m *PlayMeta) ClearRematchIfPendingBy(color engine.Color) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.RematchProposedBy != color {
		return false
	}
	m.RematchProposedBy = engine.NoColor
	m.RematchProposedAt = nil
	return true
}
