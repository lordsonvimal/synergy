package server

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/ui/components"
	"github.com/lordsonvimal/synergy/apps/chess/ui/ui_store"
	"github.com/starfederation/datastar-go/datastar"
)

// Broadcast the current selection state to the client
func broadcastSelection(c *gin.Context, g *game.Game) error {
	sse := datastar.NewSSE(c.Writer, c.Request)

	selectionSnapshot := g.SelectionSnapshot()

	b, err := json.Marshal(selectionSnapshot)
	if err != nil {
		return err
	}

	sse.PatchSignals(b)
	return nil
}

// Broadcast the updated board state to the client
func broadcastBoard(c *gin.Context, g *game.Game) error {
	sse := datastar.NewSSE(c.Writer, c.Request)

	buf := new(strings.Builder)
	components.RenderChessBoard(g).Render(c.Request.Context(), buf)

	sse.PatchElements(buf.String())

	return broadcastSelection(c, g)
}

// Show promotion UI for selecting a piece
func broadcastPromotion(c *gin.Context, signals *ui_store.ChessBoardSignals) error {
	sse := datastar.NewSSE(c.Writer, c.Request)

	b, err := json.Marshal(signals)
	if err != nil {
		return err
	}

	sse.PatchSignals(b)
	return nil
}
