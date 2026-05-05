package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/ui/components"
	"github.com/lordsonvimal/synergy/apps/chess/ui/ui_store"
	"github.com/starfederation/datastar-go/datastar"
)

// broadcastBoard sends an updated board HTML + signals patch via DataStar SSE.
// Used for per-request (non-streaming) move responses.
func broadcastBoard(c *gin.Context, g *game.Game, signals *ui_store.ChessBoardSignals) error {
	sse := datastar.NewSSE(c.Writer, c.Request)

	buf := new(strings.Builder)
	components.RenderChessBoard(g).Render(c.Request.Context(), buf)

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

// writeSSEEvent writes a single raw SSE event to a gin.ResponseWriter.
// format: "data: <payload>\n\n"
func writeSSEEvent(c *gin.Context, payload []byte) {
	fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
	c.Writer.Flush()
}
