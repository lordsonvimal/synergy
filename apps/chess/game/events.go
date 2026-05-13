package game

// ClockTickEvent is broadcast at 1 Hz by the watchdog goroutine.
// server_ts_ns lets the client compensate for transit time.
// white_running / black_running tell the client which clock is counting down.
type ClockTickEvent struct {
	Type             string `json:"type"`
	WhiteRemainingNs int64  `json:"white_remaining_ns"`
	BlackRemainingNs int64  `json:"black_remaining_ns"`
	WhiteRunning     bool   `json:"white_running"`
	BlackRunning     bool   `json:"black_running"`
	ServerTsNs       int64  `json:"server_ts_ns"`
}

// GameOverEvent is broadcast when the game ends for any reason.
type GameOverEvent struct {
	Type      string    `json:"type"`
	State     GameState `json:"state"`
	Winner    int       `json:"winner"`
	StateText string    `json:"state_text"`
}

// OnlineStatusEvent is broadcast when a player connects or disconnects via SSE.
type OnlineStatusEvent struct {
	Type        string `json:"type"`
	WhiteOnline bool   `json:"white_online"`
	BlackOnline bool   `json:"black_online"`
}

// ClockUnlockedEvent is broadcast the first time both players are simultaneously online.
type ClockUnlockedEvent struct {
	Type string `json:"type"`
}

// ClaimVictoryEvent is broadcast when a player has been disconnected for ≥60s.
type ClaimVictoryEvent struct {
	Type            string `json:"type"`
	DisconnectedFor string `json:"disconnected_for"` // "white" | "black"
}

// GameCancelledEvent is broadcast when a game is abandoned before any moves.
type GameCancelledEvent struct {
	Type string `json:"type"`
}

// RematchProposedEvent is broadcast to the opponent when a rematch is proposed.
type RematchProposedEvent struct {
	Type       string `json:"type"`
	ProposedBy string `json:"proposed_by"` // "white" | "black"
}

// RematchAcceptedEvent is broadcast to both players when a rematch is accepted.
// ProposerRedirectURL contains the token URL the proposer should navigate to,
// since the server can only set the accepter's cookie via the HTTP response.
type RematchAcceptedEvent struct {
	Type                string `json:"type"`
	ProposerRedirectURL string `json:"proposer_redirect_url"`
}

// RematchDeclinedEvent is broadcast to the proposer when their rematch is declined.
type RematchDeclinedEvent struct {
	Type string `json:"type"`
}

// RematchExpiredEvent is broadcast to both players when a rematch proposal times out.
type RematchExpiredEvent struct {
	Type string `json:"type"`
}
