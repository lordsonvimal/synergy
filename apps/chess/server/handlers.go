package server

import (
	"net/http"
	"strconv"
	"time"

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
	errMsg := gameErrorMessage(c.Query("error"))
	Render(c, http.StatusOK, pages.GameModesPage(modes, CSRFToken(c), errMsg))
}

func ShowGame(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/?error=server_error")
		return
	}

	gameID := c.Param("gameID")
	g, ok := repo.Get(gameID)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/?error=game_not_found")
		return
	}

	Render(c, http.StatusOK, pages.NewGamePage(g))
}

func CreateGame(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	logger.Info(ctx).Bool("repo found", ok).Msg("Handler: CreateGame")
	if !ok {
		c.Redirect(http.StatusSeeOther, "/?error=server_error")
		return
	}

	selectedMode := c.PostForm("mode")
	gm, err := game.FindGameModeByName(selectedMode)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?error=invalid_mode")
		return
	}

	g := game.NewGame(&gm)
	repo.Add(g)

	// PRG: redirect so reload does not trigger resubmission
	c.Redirect(http.StatusSeeOther, "/game/"+g.ID)
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

// PingHandler returns the server's current Unix nanosecond timestamp.
// Clients call this 3× at game start to compute a clock offset (NTP-style).
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"server_ns": time.Now().UnixNano()})
}

// GameEventsHandler opens a persistent SSE stream for a game.
// It delivers clock_tick events (1 Hz), board_update patches, and game_over events.
func GameEventsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	gameID := c.Param("gameID")
	g, ok := repo.Get(gameID)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable nginx buffering

	ch := g.Hub.Subscribe()
	defer g.Hub.Unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return
		case msg, open := <-ch:
			if !open {
				return
			}
			writeSSEEvent(c, msg)
		}
	}
}

// gameErrorMessage maps error query params to user-readable messages.
func gameErrorMessage(code string) string {
	switch code {
	case "invalid_mode":
		return "That game mode is no longer available. Please choose another."
	case "game_not_found":
		return "Game not found. It may have expired — please start a new game."
	case "server_error":
		return "Something went wrong on our end. Please try again."
	default:
		return ""
	}
}
