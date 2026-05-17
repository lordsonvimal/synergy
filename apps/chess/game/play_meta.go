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
	FirstMoveDeadline        *time.Time // 60s after both first connect

	// JoinDeadline is the wall-clock at which the game auto-cancels if the
	// opponent has not yet claimed their seat. Set on game creation, cleared
	// once both players have connected.
	JoinDeadline *time.Time

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

	// Draw-offer state (mid-game).
	DrawOfferedBy engine.Color
	DrawOfferedAt *time.Time

	// Takeback state (mid-game). Only proposable when it's the opponent's turn
	// (i.e. proposer has just moved); auto-declined if the opponent moves before
	// they respond.
	TakebackOfferedBy engine.Color
	TakebackOfferedAt *time.Time

	// Per-position offer locks: each colour can only offer a draw / takeback
	// once at any given position (game.Seq). -1 = never offered. Cleared on
	// any move via Seq advancing past the stored value, so the offer becomes
	// available again at the new position. Indexed [White, Black].
	LastDrawSeq     [2]int64
	LastTakebackSeq [2]int64

	// Per-color idempotency cache for move POSTs. When a client retries a
	// move (network blip, app foregrounded mid-flight, etc.) it sends the
	// same clientMoveId — the server returns the cached outcome instead of
	// re-applying. Without this a retry against a now-applied move would
	// look like a stale-baseSeq conflict and the client would needlessly
	// roll back. Indexed [White, Black].
	lastMoveID     [2]string
	lastMoveResult [2]MoveResult
	lastMoveSnap   [2]SignalsSnapshot
}

// RememberMove stores the outcome of the most recent move POST for a color so
// that a retried POST with the same clientMoveId can be answered from cache
// without re-applying. Pass an empty id to skip (clients without idempotency).
func (m *PlayMeta) RememberMove(color engine.Color, id string, result MoveResult, snap SignalsSnapshot) {
	if id == "" || (color != engine.White && color != engine.Black) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastMoveID[color] = id
	m.lastMoveResult[color] = result
	m.lastMoveSnap[color] = snap
}

// RecallMove returns the cached outcome for (color, id). The second return is
// false if no match — caller must proceed with a fresh apply.
func (m *PlayMeta) RecallMove(color engine.Color, id string) (MoveResult, SignalsSnapshot, bool) {
	if id == "" || (color != engine.White && color != engine.Black) {
		return 0, SignalsSnapshot{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastMoveID[color] != id {
		return 0, SignalsSnapshot{}, false
	}
	return m.lastMoveResult[color], m.lastMoveSnap[color], true
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
		m.JoinDeadline = nil
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

// IsSeatClaimed reports whether the given role ("white"/"black") has been
// claimed by a participant.
func (m *PlayMeta) IsSeatClaimed(role string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch role {
	case "white":
		return m.WhiteClaimed
	case "black":
		return m.BlackClaimed
	}
	return false
}

// GetJoinDeadlineNs returns the opponent-join deadline as unix nanoseconds,
// or 0 if no deadline is set (e.g. both players have already connected).
func (m *PlayMeta) GetJoinDeadlineNs() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.JoinDeadline == nil {
		return 0
	}
	return m.JoinDeadline.UnixNano()
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

// GetDrawOfferer returns the color of the player with a pending draw offer, or NoColor.
func (m *PlayMeta) GetDrawOfferer() engine.Color {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.DrawOfferedBy
}

// TryProposeDraw atomically sets a draw offer if none is pending and the
// proposer has not already offered a draw at this position (currentSeq).
func (m *PlayMeta) TryProposeDraw(proposer engine.Color, currentSeq uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DrawOfferedBy != engine.NoColor {
		return false
	}
	if m.LastDrawSeq[proposer] == int64(currentSeq) {
		return false
	}
	now := time.Now()
	m.DrawOfferedBy = proposer
	m.DrawOfferedAt = &now
	m.LastDrawSeq[proposer] = int64(currentSeq)
	return true
}

// HasOfferedDrawAt reports whether the given colour has already proposed a
// draw at the given position (Seq). Used by the UI to show / disable the
// "offer draw" button without round-tripping to attempt the request.
func (m *PlayMeta) HasOfferedDrawAt(color engine.Color, currentSeq uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.LastDrawSeq[color] == int64(currentSeq)
}

// AcceptAndClearDraw clears a draw offer if the accepter is not the proposer.
func (m *PlayMeta) AcceptAndClearDraw(accepter engine.Color) (engine.Color, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DrawOfferedBy == engine.NoColor || m.DrawOfferedBy == accepter {
		return engine.NoColor, false
	}
	proposer := m.DrawOfferedBy
	m.DrawOfferedBy = engine.NoColor
	m.DrawOfferedAt = nil
	return proposer, true
}

// ClearDraw unconditionally clears a pending draw offer.
func (m *PlayMeta) ClearDraw() {
	m.mu.Lock()
	m.DrawOfferedBy = engine.NoColor
	m.DrawOfferedAt = nil
	m.mu.Unlock()
}

// ClearDrawIfPendingBy clears the offer only if the current proposer matches color.
func (m *PlayMeta) ClearDrawIfPendingBy(color engine.Color) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DrawOfferedBy != color {
		return false
	}
	m.DrawOfferedBy = engine.NoColor
	m.DrawOfferedAt = nil
	return true
}

// GetTakebackProposer returns the color of the player with a pending takeback offer, or NoColor.
func (m *PlayMeta) GetTakebackProposer() engine.Color {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.TakebackOfferedBy
}

// TryProposeTakeback atomically sets a takeback offer if none is pending and
// the proposer has not already requested a takeback at this position.
func (m *PlayMeta) TryProposeTakeback(proposer engine.Color, currentSeq uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.TakebackOfferedBy != engine.NoColor {
		return false
	}
	if m.LastTakebackSeq[proposer] == int64(currentSeq) {
		return false
	}
	now := time.Now()
	m.TakebackOfferedBy = proposer
	m.TakebackOfferedAt = &now
	m.LastTakebackSeq[proposer] = int64(currentSeq)
	return true
}

// HasOfferedTakebackAt reports whether the given colour has already requested
// a takeback at the given position.
func (m *PlayMeta) HasOfferedTakebackAt(color engine.Color, currentSeq uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.LastTakebackSeq[color] == int64(currentSeq)
}

// AcceptAndClearTakeback clears a takeback offer if the accepter is not the proposer.
func (m *PlayMeta) AcceptAndClearTakeback(accepter engine.Color) (engine.Color, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.TakebackOfferedBy == engine.NoColor || m.TakebackOfferedBy == accepter {
		return engine.NoColor, false
	}
	proposer := m.TakebackOfferedBy
	m.TakebackOfferedBy = engine.NoColor
	m.TakebackOfferedAt = nil
	return proposer, true
}

// ClearTakeback unconditionally clears a pending takeback offer.
func (m *PlayMeta) ClearTakeback() {
	m.mu.Lock()
	m.TakebackOfferedBy = engine.NoColor
	m.TakebackOfferedAt = nil
	m.mu.Unlock()
}

// ClearTakebackIfPendingBy clears the offer only if the current proposer matches color.
func (m *PlayMeta) ClearTakebackIfPendingBy(color engine.Color) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.TakebackOfferedBy != color {
		return false
	}
	m.TakebackOfferedBy = engine.NoColor
	m.TakebackOfferedAt = nil
	return true
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
