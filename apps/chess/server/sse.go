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

// broadcastBoard sends an updated board + notation panel HTML and signals patch
// via DataStar SSE. Used for per-request (non-streaming) move responses.
func broadcastBoard(c *gin.Context, g *game.Game, signals *ui_store.ChessBoardSignals) error {
	sse := datastar.NewSSE(c.Writer, c.Request)
	ctx := c.Request.Context()

	buf := new(strings.Builder)

	// Patch chessboard.
	components.RenderChessBoard(g).Render(ctx, buf)
	sse.PatchElements(buf.String())

	// Patch move notation panel — DataStar morphs #move-notation-panel in place.
	buf.Reset()
	components.MoveNotationPanel(g.ID, g.History).Render(ctx, buf)
	sse.PatchElements(buf.String())

	return broadcastSignals(c, signals)
}

// broadcastSignals sends a DataStar signal patch via SSE.
func broadcastSignals(c *gin.Context, signals *ui_store.ChessBoardSignals) error {
	sse := datastar.NewSSE(c.Writer, c.Request)

	b, err := json.Marshal(signals)
	if err != nil {
		return err
	}

	sse.PatchSignals(b)
	return nil
}
