package server

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/starfederation/datastar-go/datastar"
)

func broadcastUpdate(c *gin.Context, g *game.Game) error {
	sse := datastar.NewSSE(c.Writer, c.Request)

	selectionSnapshot := g.SelectionSnapshot()

	b, err := json.Marshal(selectionSnapshot)
	if err != nil {
		return err
	}

	sse.PatchSignals(b)
	return nil
}
