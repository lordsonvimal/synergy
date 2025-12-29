package server

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
)

// --------------------------
// WebSocket message from client
// --------------------------
type MoveMsg struct {
	UCI string `json:"uci"` // e.g., "e2e4"
	RTT int64  `json:"rtt"` // round-trip time in nanoseconds
}

// --------------------------
// WebSocket upgrader
// --------------------------
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// --------------------------
// Active connections per game
// --------------------------
type GameConnections struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]struct{}
}

func NewGameConnections() *GameConnections {
	return &GameConnections{
		conns: make(map[*websocket.Conn]struct{}),
	}
}

func (gc *GameConnections) Add(conn *websocket.Conn) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.conns[conn] = struct{}{}
}

func (gc *GameConnections) Remove(conn *websocket.Conn) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	delete(gc.conns, conn)
}

func (gc *GameConnections) Broadcast(msg any) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	for conn := range gc.conns {
		conn.WriteJSON(msg)
	}
}

// Convert MoveStack to a slice of UCI strings
func movesFromStack(b *engine.Board) []string {
	moves := make([]string, len(b.MoveStack))
	for i, ms := range b.MoveStack {
		move := engine.Move{
			From:      ms.From,
			To:        ms.To,
			Promotion: ms.Promotion,
			Flags:     ms.Flags,
		}
		moves[i] = move.ToUCI() // assumes you have Move.ToUCI() implemented
	}
	return moves
}

// --------------------------
// Helper: create snapshot
// --------------------------
func gameSnapshot(g *game.Game, lastMove string, status string) map[string]any {
	return map[string]any{
		"seq":      g.Seq,
		"board":    g.Board.FEN(),
		"lastMove": lastMove, // for UI animation
		"status":   status,   // "ok" or "illegal"
		"clock": map[string]int64{
			"white": g.Clock.White.RemainingNs,
			"black": g.Clock.Black.RemainingNs,
		},
		"check":     g.IsCheck(),
		"checkmate": g.IsCheckmate(),
		"stalemate": g.IsStalemate(),
		"moves":     movesFromStack(g.Board), // optional PGN / move list
	}
}

// --------------------------
// Game WebSocket handler
// --------------------------
func GameWS(g *game.Game, gc *GameConnections) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Could not open websocket", http.StatusBadRequest)
			return
		}
		defer conn.Close()
		gc.Add(conn)
		defer gc.Remove(conn)

		// Send initial snapshot
		conn.WriteJSON(gameSnapshot(g, "", "ok"))

		for {
			var msg MoveMsg
			if err := conn.ReadJSON(&msg); err != nil {
				return // client disconnected
			}

			// Compute lag-compensated time (cap 150ms)
			lagNs := msg.RTT / 2
			if lagNs > 150_000_000 {
				lagNs = 150_000_000
			}

			// Parse UCI move
			move := engine.MoveFromUCI(msg.UCI)

			// Apply move
			if g.ApplyMove(move, lagNs) {
				// Legal move
				snapshot := gameSnapshot(g, msg.UCI, "ok")
				gc.Broadcast(snapshot)
			} else {
				// Illegal move, notify only this client
				conn.WriteJSON(gameSnapshot(g, msg.UCI, "illegal"))
			}
		}
	}
}
