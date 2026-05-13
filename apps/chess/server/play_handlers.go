package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lordsonvimal/synergy/apps/chess/db"
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/logger"
	"github.com/lordsonvimal/synergy/apps/chess/store"
	"github.com/lordsonvimal/synergy/apps/chess/ui/pages"
	"github.com/lordsonvimal/synergy/apps/chess/ui/ui_store"
	"github.com/starfederation/datastar-go/datastar"
)

// generateParticipantToken generates a cryptographically random 32-char hex token.
func generateParticipantToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sseBytes formats v as a raw SSE data frame.
func sseBytes(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return fmt.Appendf(nil, "data: %s\n\n", b)
}

// CreatePlay handles POST /play.
func CreatePlay(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/?error=server_error")
		return
	}

	// Block if creator already has an ongoing play game.
	if claims, err := GetPlaySession(c.Request); err == nil {
		if g, ok := repo.Get(claims.GameID); ok && g.IsOngoing() {
			c.Redirect(http.StatusSeeOther, "/?error=already_in_game")
			return
		}
	}

	selectedMode := c.PostForm("mode")
	gm, err := game.FindGameModeByName(selectedMode)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?error=invalid_mode")
		return
	}

	creatorIsWhite := time.Now().UnixNano()%2 == 0
	creatorRole := "white"
	if !creatorIsWhite {
		creatorRole = "black"
	}

	whiteToken, err := generateParticipantToken()
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?error=server_error")
		return
	}
	blackToken, err := generateParticipantToken()
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?error=server_error")
		return
	}

	meta := &game.PlayMeta{
		SessionID:          uuid.New().String(),
		WhiteParticipantID: uuid.New().String(),
		BlackParticipantID: uuid.New().String(),
		WhiteToken:         whiteToken,
		BlackToken:         blackToken,
		WhiteClaimed:       creatorIsWhite,
		BlackClaimed:       !creatorIsWhite,
		OriginalMode:       gm,
		RematchProposedBy:  engine.NoColor,
		ClaimVictoryFor:    engine.NoColor,
	}

	g := game.NewPlayGame(&gm, meta)
	repo.Add(g)

	if dbRepo, ok := store.GetDBRepoFromContext(ctx); ok {
		persistPlayGame(ctx, dbRepo, g, meta, &gm, creatorIsWhite)
	}

	if err := SetPlaySession(c, g.ID, creatorRole); err != nil {
		logger.Error(ctx).Err(err).Msg("CreatePlay: set session cookie")
	}

	c.Redirect(http.StatusSeeOther, "/play/"+g.ID)
}

// persistPlayGame creates all DB rows for a new online play game.
func persistPlayGame(
	ctx context.Context,
	dbRepo db.Repository,
	g *game.Game,
	meta *game.PlayMeta,
	mode *game.GameMode,
	creatorIsWhite bool,
) {
	now := time.Now().UnixNano()
	timeCtrlNs := mode.TimeNs
	incNs := mode.Increment

	if err := dbRepo.Sessions().Create(ctx, &db.Session{
		ID:          meta.SessionID,
		SessionType: "play",
		Status:      "pending",
		CreatedAt:   now,
	}); err != nil {
		logger.Error(ctx).Err(err).Msg("persistPlayGame: create session")
		return
	}

	if err := dbRepo.Participants().Create(ctx, &db.Participant{
		ID: meta.WhiteParticipantID, SessionID: meta.SessionID,
		Role: "white", DisplayName: "White",
		Token: meta.WhiteToken, JoinedAt: now, Claimed: creatorIsWhite,
	}); err != nil {
		logger.Error(ctx).Err(err).Msg("persistPlayGame: create white participant")
		return
	}
	if err := dbRepo.Participants().Create(ctx, &db.Participant{
		ID: meta.BlackParticipantID, SessionID: meta.SessionID,
		Role: "black", DisplayName: "Black",
		Token: meta.BlackToken, JoinedAt: now, Claimed: !creatorIsWhite,
	}); err != nil {
		logger.Error(ctx).Err(err).Msg("persistPlayGame: create black participant")
		return
	}

	wpID := meta.WhiteParticipantID
	bpID := meta.BlackParticipantID
	if err := dbRepo.Games().Create(ctx, &db.Game{
		ID:                 g.ID,
		SessionID:          meta.SessionID,
		WhiteParticipantID: &wpID,
		BlackParticipantID: &bpID,
		TimeControlNs:      &timeCtrlNs,
		IncrementNs:        &incNs,
		Variant:            mode.Variant,
		Status:             "ongoing",
		CreatedAt:          now,
	}); err != nil {
		logger.Error(ctx).Err(err).Msg("persistPlayGame: create game")
		return
	}

	if err := dbRepo.GameEvents().InsertBatch(ctx, []*db.GameEvent{{
		GameID:     g.ID,
		SessionID:  meta.SessionID,
		EventType:  "game_created",
		OccurredAt: now,
	}}); err != nil {
		logger.Error(ctx).Err(err).Msg("persistPlayGame: game_created event")
	}

	g.InitBatch(
		meta.SessionID,
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

// ShowPlayGame handles GET /play/:gameID.
func ShowPlayGame(c *gin.Context) {
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
			if restored, err := loadPlayGameFromDB(ctx, dbRepo, gameID); err == nil {
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
	if g.PlayMeta == nil {
		c.Redirect(http.StatusSeeOther, "/game/"+gameID)
		return
	}

	role, redirectURL := resolvePlayRole(c, g)
	if redirectURL != "" {
		c.Redirect(http.StatusSeeOther, redirectURL)
		return
	}

	flipped := role == "black"
	Render(c, http.StatusOK, pages.PlayGamePage(g, role, flipped, CSRFToken(c)))
}

// resolvePlayRole determines the visitor's role.
// Returns (role, "") on success or ("", redirectURL) when a claim redirect is needed.
func resolvePlayRole(c *gin.Context, g *game.Game) (role, redirectURL string) {
	// 1. Valid cookie for this game.
	if claims, err := GetPlaySession(c.Request); err == nil && claims.GameID == g.ID {
		return claims.Role, ""
	}

	// 2. Token query param — atomic claim.
	if token := c.Query("token"); token != "" {
		if participantID, claimedRole, ok := g.PlayMeta.TryClaim(token); ok {
			claimParticipant(c, g, participantID, claimedRole)
			return "", "/play/" + g.ID
		}
		// Token matched but already claimed → spectator (fall through).
	}

	return "spectator", ""
}

func claimParticipant(c *gin.Context, g *game.Game, participantID, role string) {
	ctx := c.Request.Context()
	if dbRepo, ok := store.GetDBRepoFromContext(ctx); ok {
		if err := dbRepo.Participants().SetClaimed(ctx, participantID, time.Now().UnixNano()); err != nil {
			logger.Error(ctx).Err(err).Str("participantID", participantID).Msg("claimParticipant")
		}
	}
	if err := SetPlaySession(c, g.ID, role); err != nil {
		logger.Error(ctx).Err(err).Msg("claimParticipant: set session")
	}
}

// PlayEventsHandler handles GET /play/:gameID/events.
func PlayEventsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	gameID := c.Param("gameID")
	g, ok := repo.Get(gameID)
	if !ok || g.PlayMeta == nil {
		c.Status(http.StatusNotFound)
		return
	}

	role := "spectator"
	if claims, err := GetPlaySession(c.Request); err == nil && claims.GameID == gameID {
		role = claims.Role
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Subscribe before RecordSSEConnect so this client's channel is present when
	// clock_unlocked is broadcast — the second player to connect triggers it.
	ch := g.Hub.Subscribe()
	defer g.Hub.Unsubscribe(ch)

	bothJustConnected := g.PlayMeta.RecordSSEConnect(role)
	defer func() {
		g.PlayMeta.RecordSSEDisconnect(role)
		whiteOnline, blackOnline := g.PlayMeta.OnlineStatus()
		g.Hub.BroadcastEvent(game.OnlineStatusEvent{
			Type: "online_status", WhiteOnline: whiteOnline, BlackOnline: blackOnline,
		})
	}()

	if bothJustConnected {
		g.StartPlayClock(time.Now().Add(30 * time.Second))
		g.Hub.BroadcastEvent(game.ClockUnlockedEvent{Type: "clock_unlocked"})
	}

	// Send current online status to this subscriber only.
	whiteOnline, blackOnline := g.PlayMeta.OnlineStatus()
	if raw := sseBytes(game.OnlineStatusEvent{
		Type: "online_status", WhiteOnline: whiteOnline, BlackOnline: blackOnline,
	}); raw != nil {
		c.Writer.Write(raw)
		c.Writer.Flush()
	}

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

// PlaySelectSquare handles POST /play/:gameID/select/:square.
func PlaySelectSquare(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		return
	}

	gameID, _ := c.Params.Get("gameID")
	squareParam, _ := c.Params.Get("square")
	squareUInt64, err := strconv.ParseUint(squareParam, 10, 8)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	square := uint8(squareUInt64)

	g, ok := repo.Get(gameID)
	if !ok || g.PlayMeta == nil {
		c.Status(http.StatusNotFound)
		return
	}

	claims, err := GetPlaySession(c.Request)
	if err != nil || claims.GameID != gameID || (claims.Role != "white" && claims.Role != "black") {
		c.Status(http.StatusForbidden)
		return
	}
	if !g.PlayMeta.IsStarted() {
		c.Status(http.StatusForbidden)
		return
	}

	playerColor := engine.White
	if claims.Role == "black" {
		playerColor = engine.Black
	}
	if g.Board.SideToMove != playerColor {
		c.Status(http.StatusForbidden)
		return
	}

	applySquareSelection(c, g, square)
	_ = ctx
}

// ClaimVictory handles POST /play/:gameID/claim-victory.
func ClaimVictory(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	gameID := c.Param("gameID")
	g, ok := repo.Get(gameID)
	if !ok || g.PlayMeta == nil {
		c.Status(http.StatusNotFound)
		return
	}

	claims, err := GetPlaySession(c.Request)
	if err != nil || claims.GameID != gameID || (claims.Role != "white" && claims.Role != "black") {
		c.Status(http.StatusForbidden)
		return
	}

	claimerColor := engine.White
	if claims.Role == "black" {
		claimerColor = engine.Black
	}

	claimFor := g.PlayMeta.GetClaimVictoryFor()
	if claimFor == engine.NoColor || claimFor == claimerColor {
		c.Status(http.StatusForbidden)
		return
	}

	if !g.AbandonWithResult(claimerColor) {
		c.Status(http.StatusConflict)
		return
	}
	_ = ctx
	c.Status(http.StatusNoContent)
}

// ProposeRematch handles POST /play/:gameID/rematch.
func ProposeRematch(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	gameID := c.Param("gameID")
	g, ok := repo.Get(gameID)
	if !ok || g.PlayMeta == nil {
		c.Status(http.StatusNotFound)
		return
	}

	claims, err := GetPlaySession(c.Request)
	if err != nil || claims.GameID != gameID || (claims.Role != "white" && claims.Role != "black") {
		c.Status(http.StatusForbidden)
		return
	}
	if g.IsOngoing() {
		c.Status(http.StatusConflict)
		return
	}

	proposerColor := engine.White
	if claims.Role == "black" {
		proposerColor = engine.Black
	}
	if !g.PlayMeta.TryProposeRematch(proposerColor) {
		c.Status(http.StatusConflict)
		return
	}

	g.Hub.BroadcastEvent(game.RematchProposedEvent{
		Type: "rematch_proposed", ProposedBy: claims.Role,
	})
	_ = ctx
	c.Status(http.StatusNoContent)
}

// AcceptRematch handles POST /play/:gameID/rematch/accept.
func AcceptRematch(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	gameID := c.Param("gameID")
	g, ok := repo.Get(gameID)
	if !ok || g.PlayMeta == nil {
		c.Status(http.StatusNotFound)
		return
	}

	claims, err := GetPlaySession(c.Request)
	if err != nil || claims.GameID != gameID || (claims.Role != "white" && claims.Role != "black") {
		c.Status(http.StatusForbidden)
		return
	}

	accepterColor := engine.White
	if claims.Role == "black" {
		accepterColor = engine.Black
	}

	_, ok = g.PlayMeta.AcceptAndClearRematch(accepterColor)
	if !ok {
		c.Status(http.StatusConflict)
		return
	}

	// Colors swap: old white → new black, old black → new white.
	accepterNewRole := "black" // accepter was white → becomes black
	proposerNewRole := "white" // proposer was black → becomes white
	if claims.Role == "black" {
		accepterNewRole = "white"
		proposerNewRole = "black"
	}

	newWhiteToken, _ := generateParticipantToken()
	newBlackToken, _ := generateParticipantToken()
	mode := g.PlayMeta.OriginalMode

	newMeta := &game.PlayMeta{
		SessionID:          uuid.New().String(),
		WhiteParticipantID: uuid.New().String(),
		BlackParticipantID: uuid.New().String(),
		WhiteToken:         newWhiteToken,
		BlackToken:         newBlackToken,
		WhiteClaimed:       accepterNewRole == "white",
		BlackClaimed:       accepterNewRole == "black",
		OriginalMode:       mode,
		RematchProposedBy:  engine.NoColor,
		ClaimVictoryFor:    engine.NoColor,
	}

	newGame := game.NewPlayGame(&mode, newMeta)
	repo.Add(newGame)

	if dbRepo, dbOk := store.GetDBRepoFromContext(ctx); dbOk {
		persistPlayGame(ctx, dbRepo, newGame, newMeta, &mode, accepterNewRole == "white")
	}

	// Set the accepter's cookie for the new game.
	if err := SetPlaySession(c, newGame.ID, accepterNewRole); err != nil {
		logger.Error(ctx).Err(err).Msg("AcceptRematch: set session")
	}

	// Proposer follows a token URL to claim their new role.
	var proposerToken string
	if proposerNewRole == "white" {
		proposerToken = newWhiteToken
	} else {
		proposerToken = newBlackToken
	}
	proposerURL := fmt.Sprintf("/play/%s?token=%s", newGame.ID, proposerToken)

	g.Hub.BroadcastEvent(game.RematchAcceptedEvent{
		Type:                "rematch_accepted",
		ProposerRedirectURL: proposerURL,
	})

	c.Redirect(http.StatusSeeOther, "/play/"+newGame.ID)
}

// DeclineRematch handles POST /play/:gameID/rematch/decline.
func DeclineRematch(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	gameID := c.Param("gameID")
	g, ok := repo.Get(gameID)
	if !ok || g.PlayMeta == nil {
		c.Status(http.StatusNotFound)
		return
	}

	claims, err := GetPlaySession(c.Request)
	if err != nil || claims.GameID != gameID {
		c.Status(http.StatusForbidden)
		return
	}

	g.PlayMeta.ClearRematch()
	g.Hub.BroadcastEvent(game.RematchDeclinedEvent{Type: "rematch_declined"})
	_ = ctx
	c.Status(http.StatusNoContent)
}

// AbandonPlayGame handles POST /play/:gameID/abandon.
func AbandonPlayGame(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	gameID := c.Param("gameID")
	g, ok := repo.Get(gameID)
	if !ok || g.PlayMeta == nil {
		ClearPlaySession(c)
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	claims, err := GetPlaySession(c.Request)
	if err != nil || claims.GameID != gameID {
		c.Status(http.StatusForbidden)
		return
	}

	abandonerColor := engine.White
	if claims.Role == "black" {
		abandonerColor = engine.Black
	}

	gameStarted := g.HistoryLen() > 0

	var winner engine.Color
	if gameStarted {
		if abandonerColor == engine.White {
			winner = engine.Black
		} else {
			winner = engine.White
		}
	} else {
		winner = engine.NoColor
	}

	if g.AbandonWithResult(winner) && !gameStarted {
		g.BroadcastEvent(game.GameCancelledEvent{Type: "game_cancelled"})
	}

	_ = ctx
	ClearPlaySession(c)
	c.Redirect(http.StatusSeeOther, "/")
}

var errNotPlayGame = errors.New("not a play game")

// loadPlayGameFromDB restores an online play game from the database after a server restart.
func loadPlayGameFromDB(ctx context.Context, dbRepo db.Repository, gameID string) (*game.Game, error) {
	dbGame, err := dbRepo.Games().Get(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if dbGame.WhiteParticipantID == nil || dbGame.BlackParticipantID == nil {
		return nil, errNotPlayGame
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

	clock := game.GameClock{
		White: game.Clock{RemainingNs: wRem},
		Black: game.Clock{RemainingNs: bRem},
		IncNs: incNs,
	}

	participants, err := dbRepo.Participants().ListBySession(ctx, dbGame.SessionID)
	if err != nil {
		return nil, err
	}

	var timeCtrlNs int64
	if dbGame.TimeControlNs != nil {
		timeCtrlNs = *dbGame.TimeControlNs
	}
	var restoredIncNs int64
	if dbGame.IncrementNs != nil {
		restoredIncNs = *dbGame.IncrementNs
	}

	meta := &game.PlayMeta{
		SessionID:         dbGame.SessionID,
		OriginalMode:      game.GameMode{TimeNs: timeCtrlNs, Increment: restoredIncNs, Variant: dbGame.Variant},
		RematchProposedBy: engine.NoColor,
		ClaimVictoryFor:   engine.NoColor,
	}
	for _, p := range participants {
		switch p.Role {
		case "white":
			meta.WhiteParticipantID = p.ID
			meta.WhiteToken = p.Token
			meta.WhiteClaimed = p.Claimed
		case "black":
			meta.BlackParticipantID = p.ID
			meta.BlackToken = p.Token
			meta.BlackClaimed = p.Claimed
		}
	}

	restored := game.NewRestoredPlayGame(gameID, board, clock, history, seq, state, winner, meta)

	if state == game.GameOngoing {
		restored.InitBatch(
			dbGame.SessionID,
			func(batchCtx context.Context, moves []game.PendingMove, events []game.PendingEvent) error {
				dbMovesBatch := make([]*db.Move, len(moves))
				for i, m := range moves {
					dbMovesBatch[i] = &db.Move{
						GameID: m.GameID, SessionID: m.SessionID, Seq: m.Seq,
						UCI: m.UCI, SAN: m.SAN, FEN: m.FEN, MoveNumber: m.MoveNumber,
						Color: m.Color, WRemNs: m.WRemNs, BRemNs: m.BRemNs,
						LagCompNs: m.LagCompNs, ThinkTimeNs: m.ThinkTimeNs, PlayedAt: m.PlayedAt,
					}
				}
				dbEventsBatch := make([]*db.GameEvent, len(events))
				for i, e := range events {
					dbEventsBatch[i] = &db.GameEvent{
						GameID: e.GameID, SessionID: e.SessionID,
						EventType: e.EventType, Payload: e.Payload, OccurredAt: e.OccurredAt,
					}
				}
				if err := dbRepo.Moves().InsertBatch(batchCtx, dbMovesBatch); err != nil {
					return err
				}
				return dbRepo.GameEvents().InsertBatch(batchCtx, dbEventsBatch)
			},
			func(batchCtx context.Context, status, winner string) error {
				endedAt := time.Now().UnixNano()
				var winnerPtr *string
				if winner != "" {
					winnerPtr = &winner
				}
				return dbRepo.Games().UpdateStatus(batchCtx, gameID, status, winnerPtr, &endedAt)
			},
		)
	}

	return restored, nil
}

// applySquareSelection contains the shared square-selection logic for both solo and play modes.
func applySquareSelection(c *gin.Context, g *game.Game, square uint8) {
	ctx := c.Request.Context()

	signals := ui_store.NewChessBoardSignals()
	datastar.ReadSignals(c.Request, signals)

	if g.HasSelection() && g.IsTarget(square) {
		move := engine.Move{From: g.GetSelectionFrom(), To: square, Promotion: engine.NoPiece}
		promoteWithPiece := signals.Promotion && signals.PromotionPiece != engine.NoPiece

		if g.IsPromotionMove(move) && !promoteWithPiece {
			signals.EnablePromotion(square)
			if err := broadcastSignals(c, signals); err != nil {
				logger.Error(ctx).Err(err).Msg("applySquareSelection: broadcast promotion")
			}
			return
		}
		if promoteWithPiece {
			move.Promotion = signals.PromotionPiece
		}
		if g.ApplyMove(move, 0) {
			signals.UpdateFromGame(g)
			if promoteWithPiece {
				signals.ClearPromotion()
			}
			if err := broadcastBoard(c, g, signals); err != nil {
				logger.Error(ctx).Err(err).Msg("applySquareSelection: broadcast board")
			}
			// Push the updated board to all other subscribers (opponent, spectators).
			if g.PlayMeta != nil {
				hubBroadcastBoard(ctx, g, signals)
			}
			return
		}
	}

	g.SelectSquare(ctx, square)
	signals.UpdateFromGame(g)
	if err := broadcastSignals(c, signals); err != nil {
		logger.Error(ctx).Err(err).Msg("applySquareSelection: broadcast selection")
	}
}
