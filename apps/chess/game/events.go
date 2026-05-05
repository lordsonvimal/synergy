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
