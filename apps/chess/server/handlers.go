package server

import (
	"io"
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
			broadcastUpdate(g)
			return
		}
	}

	logger.Info(ctx).Uint8("selecting square", square).Msg("Selecting Square")
	g.SelectSquare(ctx, square)

	broadcastUpdate(g)
}

func LiveChessUpdates(c *gin.Context) {
	gameID := c.Param("gameID")

	// 1. Set SSE Headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Essential for Nginx proxies

	// 2. Create a client channel
	// We use a buffered channel to prevent slow clients from blocking the broadcaster
	clientChan := make(chan string, 10)

	// 3. Register this client
	streamsMu.Lock()
	gameStreams[gameID] = append(gameStreams[gameID], clientChan)
	streamsMu.Unlock()

	// 4. Cleanup on disconnect
	defer func() {
		streamsMu.Lock()
		defer streamsMu.Unlock()

		// Remove this specific channel from the slice
		listeners := gameStreams[gameID]
		for i, ch := range listeners {
			if ch == clientChan {
				gameStreams[gameID] = append(listeners[:i], listeners[i+1:]...)
				break
			}
		}
		close(clientChan)
	}()

	// 5. Main Event Loop
	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-clientChan:
			if !ok {
				return false // Channel closed
			}
			// Write the raw Datastar fragment to the stream
			c.Writer.WriteString(msg)
			return true
		case <-c.Request.Context().Done():
			return false // Client disconnected
		}
	})
}
