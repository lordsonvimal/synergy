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
	"github.com/lordsonvimal/synergy/apps/chess/ui/ui_store"
	"github.com/starfederation/datastar-go/datastar"
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
	logger.Info(ctx).Bool("game found", ok).Uint8("square", square).Msg("Handler: SelectSquare - Get Game Square")
	if !ok {
		return
	}

	signals := ui_store.NewChessBoardSignals()
	datastar.ReadSignals(c.Request, signals)

	logger.Info(ctx).Bool("has selection", g.HasSelection()).Uint8("isTarget", square).Msg("Handler: Moving Piece")
	if g.HasSelection() && g.IsTarget(square) {
		move := engine.Move{From: g.GetSelectionFrom(), To: square, Promotion: engine.NoPiece}
		promoteWithPiece := signals.Promotion && signals.PromotionPiece != engine.NoPiece

		// Is the selected move a promotion?
		if g.IsPromotionMove(move) && !promoteWithPiece {
			signals.EnablePromotion(square)

			err := broadcastSignals(c, signals)
			if err != nil {
				logger.Error(ctx).Err(err).Msg("Failed to broadcast promotion signal")
			}
			return
		}

		// Update move with promotion piece if already selected
		if promoteWithPiece {
			move.Promotion = signals.PromotionPiece
		}
		if g.ApplyMove(move, 0) {
			signals.UpdateFromGame(g)
			if promoteWithPiece {
				signals.ClearPromotion()
			}

			err := broadcastBoard(c, g, signals)
			if err != nil {
				logger.Error(ctx).Err(err).Msg("Failed to broadcast board update")
			}
			return

		} else {
			logger.Info(ctx).Msg("Invalid move attempted")
		}
	}

	logger.Info(ctx).Uint8("selecting square", square).Msg("Selecting Square")
	g.SelectSquare(ctx, square)
	signals.UpdateFromGame(g)
	err = broadcastSignals(c, signals)
	if err != nil {
		logger.Error(ctx).Err(err).Msg("Failed to broadcast selection update")
	}
}
