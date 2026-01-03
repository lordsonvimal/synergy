package server

import (
	"net/http"
	"slices"

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

func SelectSquare(c *gin.Context) {
	game := MustGame(c)
	square := parseSquare(c)

	// MOVE phase
	if game.Selection.FromSquare != nil &&
		slices.Contains(game.Selection.Targets, square) {

		game.Engine.MakeMove(*game.Selection.FromSquare, square)
		game.ClearSelection()

		game.BroadcastBoard()
		return
	}

	// SELECTION phase
	piece := game.Board.PieceAt(square)
	if piece == nil {
		game.ClearSelection()
		game.BroadcastSelection()
		return
	}

	targets := game.Engine.LegalMoves(square)
	game.Selection.FromSquare = &square
	game.Selection.Targets = targets
	game.BroadcastSelection()
}

func LiveChessUpdates(c *gin.Context) {

}
