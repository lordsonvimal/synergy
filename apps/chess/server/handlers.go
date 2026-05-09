package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lordsonvimal/synergy/apps/chess/db"
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/logger"
	"github.com/lordsonvimal/synergy/apps/chess/store"
	"github.com/lordsonvimal/synergy/apps/chess/ui/components"
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
		if dbRepo, dbOk := store.GetDBRepoFromContext(ctx); dbOk {
			if restored, err := loadGameFromDB(ctx, dbRepo, gameID); err == nil {
				repo.Add(restored)
				g = restored
				ok = true
			}
		}
	}
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

	// Wire up DB persistence if available.
	if dbRepo, ok := store.GetDBRepoFromContext(ctx); ok {
		persistGame(ctx, dbRepo, g, &gm)
	}

	// PRG: redirect so reload does not trigger resubmission
	c.Redirect(http.StatusSeeOther, "/game/"+g.ID)
}

// persistGame creates the session + game rows and initialises the move batch.
func persistGame(ctx context.Context, dbRepo db.Repository, g *game.Game, mode *game.GameMode) {
	now := time.Now().UnixNano()
	sessionID := uuid.New().String()

	timeCtrlNs := mode.TimeNs
	incNs := mode.Increment

	if err := dbRepo.Sessions().Create(ctx, &db.Session{
		ID:          sessionID,
		SessionType: "play",
		Status:      "active",
		CreatedAt:   now,
		StartedAt:   &now,
	}); err != nil {
		logger.Error(ctx).Err(err).Msg("persistGame: create session")
		return
	}

	if err := dbRepo.Games().Create(ctx, &db.Game{
		ID:            g.ID,
		SessionID:     sessionID,
		TimeControlNs: &timeCtrlNs,
		IncrementNs:   &incNs,
		Variant:       mode.Variant,
		Status:        "ongoing",
		CreatedAt:     now,
	}); err != nil {
		logger.Error(ctx).Err(err).Msg("persistGame: create game")
		return
	}

	startEvent := &db.GameEvent{
		GameID:     g.ID,
		SessionID:  sessionID,
		EventType:  "game_started",
		OccurredAt: now,
	}
	if err := dbRepo.GameEvents().InsertBatch(ctx, []*db.GameEvent{startEvent}); err != nil {
		logger.Error(ctx).Err(err).Msg("persistGame: game_started event")
	}

	g.InitBatch(
		sessionID,
		func(batchCtx context.Context, moves []game.PendingMove, events []game.PendingEvent) error {
			dbMoves := make([]*db.Move, len(moves))
			for i, m := range moves {
				dbMoves[i] = &db.Move{
					GameID:      m.GameID,
					SessionID:   m.SessionID,
					Seq:         m.Seq,
					UCI:         m.UCI,
					SAN:         m.SAN,
					FEN:         m.FEN,
					MoveNumber:  m.MoveNumber,
					Color:       m.Color,
					WRemNs:      m.WRemNs,
					BRemNs:      m.BRemNs,
					LagCompNs:   m.LagCompNs,
					ThinkTimeNs: m.ThinkTimeNs,
					PlayedAt:    m.PlayedAt,
				}
			}
			dbEvents := make([]*db.GameEvent, len(events))
			for i, e := range events {
				dbEvents[i] = &db.GameEvent{
					GameID:     e.GameID,
					SessionID:  e.SessionID,
					EventType:  e.EventType,
					Payload:    e.Payload,
					OccurredAt: e.OccurredAt,
				}
			}
			if err := dbRepo.Moves().InsertBatch(batchCtx, dbMoves); err != nil {
				return err
			}
			return dbRepo.GameEvents().InsertBatch(batchCtx, dbEvents)
		},
		func(batchCtx context.Context, status, winner string) error {
			endedAt := time.Now().UnixNano()
			var winnerPtr *string
			if winner != "" {
				winnerPtr = &winner
			}
			return dbRepo.Games().UpdateStatus(batchCtx, g.ID, status, winnerPtr, &endedAt)
		},
	)
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
			// Hub messages are pre-formatted SSE (named or unnamed events).
			c.Writer.Write(msg)
			c.Writer.Flush()
		}
	}
}

// historyNavSignals is the signal payload patched by history navigation handlers.
type historyNavSignals struct {
	HistoryIdx     int  `json:"historyIdx"`
	ViewingHistory bool `json:"viewingHistory"`
}

// BoardAtHistoryHandler handles GET /game/:gameID/board-at/:halfMoveIdx.
// It renders the board position after the given 0-based half-move index and
// patches both the history board content and the navigation signals.
// The special value "live" returns to the live board.
func BoardAtHistoryHandler(c *gin.Context) {
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

	idxParam := c.Param("halfMoveIdx")

	sse := datastar.NewSSE(c.Writer, c.Request)

	// "live" is the sentinel value for returning to the live board.
	if idxParam == "live" {
		b, _ := json.Marshal(historyNavSignals{HistoryIdx: -1, ViewingHistory: false})
		sse.PatchSignals(b)
		return
	}

	idx64, err := strconv.ParseInt(idxParam, 10, 64)
	if err != nil || idx64 < 0 {
		c.Status(http.StatusBadRequest)
		return
	}
	idx := int(idx64)

	fen, found := g.HistoryFENAt(idx)
	if !found {
		c.Status(http.StatusNotFound)
		return
	}

	board, err := engine.BoardFromFENDisplay(fen)
	if err != nil {
		logger.Error(ctx).Err(err).Str("fen", fen).Msg("BoardAtHistoryHandler: FEN parse error")
		c.Status(http.StatusInternalServerError)
		return
	}

	buf := new(strings.Builder)
	components.RenderHistoryBoard(board).Render(ctx, buf)
	sse.PatchElements(buf.String())

	b, _ := json.Marshal(historyNavSignals{HistoryIdx: idx, ViewingHistory: true})
	sse.PatchSignals(b)
}

// NavigateHistoryHandler handles POST /game/:gameID/history/navigate.
// It reads the current $historyIdx from the DataStar signal payload and the
// ?direction query param (+1 or -1) to compute the next position, then
// delegates to BoardAtHistoryHandler logic.
func NavigateHistoryHandler(c *gin.Context) {
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

	dir, err := strconv.Atoi(c.Query("direction"))
	if err != nil || (dir != 1 && dir != -1) {
		c.Status(http.StatusBadRequest)
		return
	}

	// Read current signal state from the request body.
	type navInput struct {
		HistoryIdx     int  `json:"historyIdx"`
		ViewingHistory bool `json:"viewingHistory"`
	}
	sig := &navInput{HistoryIdx: -1}
	datastar.ReadSignals(c.Request, sig)

	total := g.HistoryLen()
	sse := datastar.NewSSE(c.Writer, c.Request)

	// Determine target index based on current state.
	// historyIdx=-1 + viewingHistory=false → live
	// historyIdx=-1 + viewingHistory=true  → initial board position (before any moves)
	// historyIdx=N                         → position after half-move N
	var targetIdx int
	switch {
	case !sig.ViewingHistory:
		// At live the current position IS after half-move total-1, so going to
		// total-1 would show the same board. Jump to total-2 so the board changes.
		// When total==1 this yields -1, which the block below turns into the
		// initial board position.
		if dir == -1 && total > 0 {
			targetIdx = total - 2
		} else {
			return
		}
	case sig.HistoryIdx == -1:
		// At initial position: next goes to first half-move, prev is a no-op.
		if dir == 1 {
			if total == 1 {
				targetIdx = total
			} else {
				targetIdx = 0
			}
		} else {
			return
		}
	default:
		if dir == 1 && sig.HistoryIdx == total-2 {
			targetIdx = total
		} else {
			targetIdx = sig.HistoryIdx + dir
		}
	}

	// Past the end → return to live.
	if targetIdx >= total {
		b, _ := json.Marshal(historyNavSignals{HistoryIdx: -1, ViewingHistory: false})
		sse.PatchSignals(b)
		return
	}

	// Before the start → show initial board position.
	if targetIdx < 0 {
		initBoard := engine.NewBoard()
		buf := new(strings.Builder)
		components.RenderHistoryBoard(initBoard).Render(ctx, buf)
		sse.PatchElements(buf.String())
		b, _ := json.Marshal(historyNavSignals{HistoryIdx: -1, ViewingHistory: true})
		sse.PatchSignals(b)
		return
	}

	fen, found := g.HistoryFENAt(targetIdx)
	if !found {
		c.Status(http.StatusNotFound)
		return
	}

	board, err := engine.BoardFromFENDisplay(fen)
	if err != nil {
		logger.Error(ctx).Err(err).Str("fen", fen).Msg("NavigateHistoryHandler: FEN parse error")
		c.Status(http.StatusInternalServerError)
		return
	}

	buf := new(strings.Builder)
	components.RenderHistoryBoard(board).Render(ctx, buf)
	sse.PatchElements(buf.String())

	b, _ := json.Marshal(historyNavSignals{HistoryIdx: targetIdx, ViewingHistory: true})
	sse.PatchSignals(b)
}

func loadGameFromDB(ctx context.Context, dbRepo db.Repository, id string) (*game.Game, error) {
	dbGame, err := dbRepo.Games().Get(ctx, id)
	if err != nil {
		return nil, err
	}
	dbMoves, err := dbRepo.Moves().ListByGame(ctx, id)
	if err != nil {
		return nil, err
	}

	var board *engine.Board
	if len(dbMoves) > 0 {
		board, err = engine.BoardFromFEN(dbMoves[len(dbMoves)-1].FEN)
		if err != nil {
			return nil, err
		}
	} else {
		board = engine.NewBoard()
	}

	history := make([]game.MoveRecord, len(dbMoves))
	for i, m := range dbMoves {
		color := engine.White
		if m.Color == "black" {
			color = engine.Black
		}
		history[i] = game.MoveRecord{SAN: m.SAN, FEN: m.FEN, Color: color, MoveNumber: m.MoveNumber}
	}

	var wRem, bRem, incNs int64
	if dbGame.TimeControlNs != nil {
		wRem, bRem = *dbGame.TimeControlNs, *dbGame.TimeControlNs
	}
	if dbGame.IncrementNs != nil {
		incNs = *dbGame.IncrementNs
	}
	if len(dbMoves) > 0 {
		last := dbMoves[len(dbMoves)-1]
		wRem, bRem = last.WRemNs, last.BRemNs
	}

	state := dbStatusToState(dbGame.Status)
	winner := engine.NoColor
	if dbGame.Winner != nil {
		switch *dbGame.Winner {
		case "white":
			winner = engine.White
		case "black":
			winner = engine.Black
		}
	}

	var seq uint64
	if len(dbMoves) > 0 {
		seq = dbMoves[len(dbMoves)-1].Seq
	}

	clock := game.GameClock{
		White: game.Clock{RemainingNs: wRem},
		Black: game.Clock{RemainingNs: bRem},
		IncNs: incNs,
	}
	return game.NewRestoredGame(id, board, clock, history, seq, state, winner), nil
}

func dbStatusToState(s string) game.GameState {
	switch s {
	case "ongoing":
		return game.GameOngoing
	case "checkmate":
		return game.GameCheckmate
	case "resigned":
		return game.GameResigned
	case "clock_flagged":
		return game.GameClockFlagged
	case "draw_stalemate":
		return game.GameDrawStalemate
	case "draw_fifty_move":
		return game.GameDrawFiftyMove
	case "draw_agreement":
		return game.GameDrawAgreement
	case "draw_threefold":
		return game.GameDrawThreefoldRepetition
	case "draw_insufficient":
		return game.GameDrawInsufficientMaterial
	default:
		return game.GameAbandoned
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
