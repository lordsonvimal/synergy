package server

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/ui/components"
	"github.com/starfederation/datastar-go/datastar"
)

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

func broadcastBoard(c *gin.Context, g *game.Game) error {
	sse := datastar.NewSSE(c.Writer, c.Request)

	buf := new(strings.Builder)
	components.RenderChessBoard(g).Render(c.Request.Context(), buf)

	sse.PatchElements(
		`<div id="chessboardcontainer">` + buf.String() + `</div>`,
	)
	return nil
}
