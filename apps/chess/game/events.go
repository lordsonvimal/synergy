package game

// ClockTickEvent carries one snapshot of clock state. Constructed by the
// watchdog and by the post-move helper, then converted to a datastar signal
// patch by the broadcast helpers — never serialised to the wire as JSON.
type ClockTickEvent struct {
	WhiteRemainingNs int64
	BlackRemainingNs int64
	WhiteRunning     bool
	BlackRunning     bool
	ServerTsNs       int64
}
