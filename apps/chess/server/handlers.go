package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/logger"
	"github.com/lordsonvimal/synergy/apps/chess/store"
	"github.com/lordsonvimal/synergy/apps/chess/ui/pages"
)

func ShowGameModes(c *gin.Context) {
	modes := game.ListGameModes()
	Render(c, http.StatusOK, pages.GameModesPage(modes))
}

func CreateGame(c *gin.Context) {
	repo, ok := store.GetRepoFromContext(c.Request.Context())
	logger.Info(c.Request.Context()).Bool("repo found", ok).Msg("Handler: CreateGame")
	if !ok {
		return
	}

	selectedMode := c.PostForm("mode")
	gm, err := game.FindGameModeByName(selectedMode)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid game mode")
		return
	}

	g := game.NewGame(&gm)
	repo.Add(g)

	// For simplicity, you can store in-memory or use session/DB
	// For now, render the chessboard page
	Render(c, http.StatusOK, pages.NewGamePage(g))
}

func SelectSquare(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	logger.Info(ctx).Bool("repo found", ok).Msg("Handler: SelectSquare")
	if !ok {
		return
	}

	gameID, ok := c.Params.Get("gameID")
	if !ok {
		logger.Error(ctx).Str("gameID found", gameID).Msg("GameID")
		return
	}

	squareParam, ok := c.Params.Get("square")
	if !ok {
		logger.Info(ctx).Str("square", squareParam).Msg("Invalid Square")
		return
	}

	squareUInt64, err := strconv.ParseUint(squareParam, 10, 8)
	if err != nil {
		logger.Info(ctx).Err(err).Str("square", squareParam).Msg("Parsing Square")
		return
	}

	square := uint8(squareUInt64)

	g, ok := repo.Get(gameID)
	logger.Info(ctx).Bool("game found", ok).Msg("Handler: SelectSquare - Get Game")
	if !ok {
		return
	}

	if g.HasSelection() && g.IsTarget(square) {
		move := engine.Move{From: g.GetSelectionFrom(), To: square}
		if g.ApplyMove(move, 0) {
			g.ClearSelection()
			err := broadcastBoard(c, g)
			if err != nil {
				logger.Error(ctx).Err(err).Msg("Failed to broadcast board update")
			}
			return
		}
	}

	logger.Info(ctx).Uint8("selecting square", square).Msg("Selecting Square")
	g.SelectSquare(ctx, square)

	err = broadcastSelection(c, g)
	if err != nil {
		logger.Error(ctx).Err(err).Msg("Failed to broadcast selection update")
	}
}
