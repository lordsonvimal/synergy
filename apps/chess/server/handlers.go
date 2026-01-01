package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/ui/pages"
)

func ShowGameModes(c *gin.Context) {
	modes := game.ListGameModes()
	Render(c, http.StatusOK, pages.GameModesPage(modes))
}

func CreateGame(c *gin.Context) {
	selectedMode := c.PostForm("mode")
	gm, err := game.FindGameModeByName(selectedMode)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid game mode")
		return
	}

	g := game.NewGame(&gm)

	// For simplicity, you can store in-memory or use session/DB
	// For now, render the chessboard page
	Render(c, http.StatusOK, pages.NewGamePage(g))
}

func LiveChessUpdates(c *gin.Context) {

}
