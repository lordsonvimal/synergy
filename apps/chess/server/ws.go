package server

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
)

// --------------------------
// WebSocket message
// --------------------------
type MoveMsg struct {
	UCI string `json:"uci"`
	RTT int64  `json:"rtt"` // round-trip time in nanoseconds
}

// --------------------------
// WebSocket upgrader
// --------------------------
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // allow all origins
}

// --------------------------
// Helper functions
// --------------------------
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// --------------------------
// Convert UCI string to Move
// --------------------------
func parseUCI(uci string) engine.Move {
	return engine.MoveFromUCI(uci)
}

// --------------------------
// Convert game state to snapshot for client
// --------------------------
func gameSnapshot(g *game.Game) map[string]interface{} {
	return map[string]interface{}{
		"seq":   g.Seq,
		"board": g.Board.String(), // assuming you have Board.String() returning FEN or visual board
		"clock": map[string]int64{
			"white": g.Clock.White.RemainingNs,
			"black": g.Clock.Black.RemainingNs,
		},
	}
}

// --------------------------
// WebSocket handler
// --------------------------
func GameWS(g *game.Game) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Could not open websocket", http.StatusBadRequest)
			return
		}
		defer conn.Close()

		for {
			var msg MoveMsg
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}

			// Compute lag-compensated latency (cap 150ms)
			lagNs := min(msg.RTT/2, 150_000_000)

			move := parseUCI(msg.UCI)

			if g.ApplyMove(move, lagNs) {
				// Send snapshot back to client
				if err := conn.WriteJSON(gameSnapshot(g)); err != nil {
					return
				}
			}
		}
	}
}
