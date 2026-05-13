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
	"github.com/starfederation/datastar-go/datastar"
)

func ShowGameModes(c *gin.Context) {
	ctx := c.Request.Context()
	errMsg := gameErrorMessage(c.Query("error"))

	var existingGameID, existingGameRole string
	if claims, err := GetPlaySession(c.Request); err == nil {
		if repo, ok := store.GetRepoFromContext(ctx); ok {
			if g, ok := repo.Get(claims.GameID); ok && g.IsOngoing() {
				existingGameID = claims.GameID
				existingGameRole = claims.Role
			}
		}
	}

	Render(c, http.StatusOK, pages.GameModesPage(
		game.ListOnlineModeGroups(),
		game.UnlimitedMode(),
		CSRFToken(c),
		errMsg,
		existingGameID,
		existingGameRole,
	))
}

func ShowSolo(c *gin.Context) {
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
			if restored, err := loadSoloGameFromDB(ctx, dbRepo, gameID); err == nil {
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

	Render(c, http.StatusOK, pages.SoloGamePage(g))
}

func CreateSolo(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
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

	if dbRepo, ok := store.GetDBRepoFromContext(ctx); ok {
		persistSoloGame(ctx, dbRepo, g, &gm)
	}

	c.Redirect(http.StatusSeeOther, "/solo/"+g.ID)
}

func persistSoloGame(ctx context.Context, dbRepo db.Repository, g *game.Game, mode *game.GameMode) {
	now := time.Now().UnixNano()
	sessionID := uuid.New().String()
	timeCtrlNs := mode.TimeNs
	incNs := mode.Increment

	if err := dbRepo.Sessions().Create(ctx, &db.Session{
		ID:          sessionID,
		SessionType: "solo",
		Status:      "active",
		CreatedAt:   now,
		StartedAt:   &now,
	}); err != nil {
		logger.Error(ctx).Err(err).Msg("persistSoloGame: create session")
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
		logger.Error(ctx).Err(err).Msg("persistSoloGame: create game")
		return
	}

	if err := dbRepo.GameEvents().InsertBatch(ctx, []*db.GameEvent{{
		GameID:     g.ID,
		SessionID:  sessionID,
		EventType:  "game_started",
		OccurredAt: now,
	}}); err != nil {
		logger.Error(ctx).Err(err).Msg("persistSoloGame: game_started event")
	}

	initGameBatch(g, sessionID, dbRepo)
}

func SoloSelectSquare(c *gin.Context) {
	repo, ok := store.GetRepoFromContext(c.Request.Context())
	if !ok {
		return
	}

	squareUInt64, err := strconv.ParseUint(c.Param("square"), 10, 8)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	g, ok := repo.Get(c.Param("gameID"))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	applySquareSelection(c, g, uint8(squareUInt64))
}

// PingHandler returns the server's current Unix nanosecond timestamp.
// Clients call this 3× at game start to compute a clock offset (NTP-style).
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"server_ns": time.Now().UnixNano()})
}

// SoloEventsHandler opens a persistent SSE stream for a solo game.
// It delivers clock_tick, board_update, and game_over events via the hub.
func SoloEventsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	g, ok := repo.Get(c.Param("gameID"))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

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

// BoardAtHistoryHandler handles GET /{solo|play}/:gameID/board-at/:halfMoveIdx.
// Shared between solo and play routes — gameID lookup works for both.
func BoardAtHistoryHandler(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	g, ok := repo.Get(c.Param("gameID"))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	idxParam := c.Param("halfMoveIdx")
	sse := datastar.NewSSE(c.Writer, c.Request)

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

	fen, found := g.HistoryFENAt(int(idx64))
	if !found {
		c.Status(http.StatusNotFound)
		return
	}

	board, err := engine.BoardFromFENDisplay(fen)
	if err != nil {
		logger.Error(ctx).Err(err).Str("fen", fen).Msg("BoardAtHistoryHandler: FEN parse")
		c.Status(http.StatusInternalServerError)
		return
	}

	buf := new(strings.Builder)
	components.RenderHistoryBoard(board).Render(ctx, buf)
	sse.PatchElements(buf.String())

	b, _ := json.Marshal(historyNavSignals{HistoryIdx: int(idx64), ViewingHistory: true})
	sse.PatchSignals(b)
}

// NavigateHistoryHandler handles POST /{solo|play}/:gameID/history/navigate.
// Shared between solo and play routes.
func NavigateHistoryHandler(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	g, ok := repo.Get(c.Param("gameID"))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	dir, err := strconv.Atoi(c.Query("direction"))
	if err != nil || (dir != 1 && dir != -1) {
		c.Status(http.StatusBadRequest)
		return
	}

	type navInput struct {
		HistoryIdx     int  `json:"historyIdx"`
		ViewingHistory bool `json:"viewingHistory"`
	}
	sig := &navInput{HistoryIdx: -1}
	datastar.ReadSignals(c.Request, sig)

	total := g.HistoryLen()
	sse := datastar.NewSSE(c.Writer, c.Request)

	var targetIdx int
	switch {
	case !sig.ViewingHistory:
		if dir == -1 && total > 0 {
			targetIdx = total - 2
		} else {
			return
		}
	case sig.HistoryIdx == -1:
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

	if targetIdx >= total {
		b, _ := json.Marshal(historyNavSignals{HistoryIdx: -1, ViewingHistory: false})
		sse.PatchSignals(b)
		return
	}

	if targetIdx < 0 {
		buf := new(strings.Builder)
		components.RenderHistoryBoard(engine.NewBoard()).Render(ctx, buf)
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
		logger.Error(ctx).Err(err).Str("fen", fen).Msg("NavigateHistoryHandler: FEN parse")
		c.Status(http.StatusInternalServerError)
		return
	}

	buf := new(strings.Builder)
	components.RenderHistoryBoard(board).Render(ctx, buf)
	sse.PatchElements(buf.String())

	b, _ := json.Marshal(historyNavSignals{HistoryIdx: targetIdx, ViewingHistory: true})
	sse.PatchSignals(b)
}

// ── Shared DB helpers ────────────────────────────────────────────────────────

// gameCore holds the fields common to both solo and play game restoration.
type gameCore struct {
	dbGame  *db.Game
	board   *engine.Board
	history []game.MoveRecord
	clock   game.GameClock
	state   game.GameState
	winner  engine.Color
	seq     uint64
}

// loadGameCore fetches a game and its moves from the DB and reconstructs the
// shared fields used by both loadSoloGameFromDB and loadPlayGameFromDB.
func loadGameCore(ctx context.Context, dbRepo db.Repository, gameID string) (*gameCore, error) {
	dbGame, err := dbRepo.Games().Get(ctx, gameID)
	if err != nil {
		return nil, err
	}
	dbMoves, err := dbRepo.Moves().ListByGame(ctx, gameID)
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

	return &gameCore{
		dbGame:  dbGame,
		board:   board,
		history: history,
		clock:   game.GameClock{White: game.Clock{RemainingNs: wRem}, Black: game.Clock{RemainingNs: bRem}, IncNs: incNs},
		state:   state,
		winner:  winner,
		seq:     seq,
	}, nil
}

func loadSoloGameFromDB(ctx context.Context, dbRepo db.Repository, id string) (*game.Game, error) {
	core, err := loadGameCore(ctx, dbRepo, id)
	if err != nil {
		return nil, err
	}
	return game.NewRestoredGame(id, core.board, core.clock, core.history, core.seq, core.state, core.winner), nil
}

// initGameBatch wires up the DB flush callbacks shared by solo and play games.
func initGameBatch(g *game.Game, sessionID string, dbRepo db.Repository) {
	g.InitBatch(
		sessionID,
		func(batchCtx context.Context, moves []game.PendingMove, events []game.PendingEvent) error {
			dbMoves := make([]*db.Move, len(moves))
			for i, m := range moves {
				dbMoves[i] = &db.Move{
					GameID: m.GameID, SessionID: m.SessionID, Seq: m.Seq,
					UCI: m.UCI, SAN: m.SAN, FEN: m.FEN, MoveNumber: m.MoveNumber,
					Color: m.Color, WRemNs: m.WRemNs, BRemNs: m.BRemNs,
					LagCompNs: m.LagCompNs, ThinkTimeNs: m.ThinkTimeNs, PlayedAt: m.PlayedAt,
				}
			}
			dbEvents := make([]*db.GameEvent, len(events))
			for i, e := range events {
				dbEvents[i] = &db.GameEvent{
					GameID: e.GameID, SessionID: e.SessionID,
					EventType: e.EventType, Payload: e.Payload, OccurredAt: e.OccurredAt,
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

func gameErrorMessage(code string) string {
	switch code {
	case "invalid_mode":
		return "That game mode is no longer available. Please choose another."
	case "game_not_found":
		return "Game not found. It may have expired — please start a new game."
	case "server_error":
		return "Something went wrong on our end. Please try again."
	case "already_in_game":
		return "You already have an ongoing game. Rejoin or abandon it before starting a new one."
	default:
		return ""
	}
}
