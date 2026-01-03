package server

import (
	"context"
	"strings"
	"sync"

	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/ui/components"
)

// A simple registry to keep track of active SSE streams
var (
	streamsMu   sync.RWMutex
	gameStreams = make(map[string][]chan string)
)

func broadcastUpdate(g *game.Game) {
	// 1. Render the Templ component to a buffer
	// We pass the game object which now contains the SelectionState
	buf := new(strings.Builder)
	components.RenderChessBoard(g).Render(context.Background(), buf)
	html := buf.String()

	// 2. Format as a Datastar Fragment
	// This tells Datastar: "Find the element with this ID and replace its innerHTML"
	sseEvent := "event: datastar-fragment\ndata: #chessboard innerHTML\n\n" + html + "\n\n"

	// 3. Send to all listeners for this specific game
	streamsMu.RLock()
	listeners := gameStreams[g.ID]
	streamsMu.RUnlock()

	for _, ch := range listeners {
		// Non-blocking send to prevent one slow client from hanging the server
		select {
		case ch <- sseEvent:
		default:
		}
	}
}
