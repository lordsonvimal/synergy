package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
)

// generateParticipantToken generates a cryptographically random 32-char hex token.
func generateParticipantToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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

	initGameBatch(g, meta.SessionID, dbRepo)
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
		c.Redirect(http.StatusSeeOther, "/solo/"+gameID)
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
//
// Two concurrent SSE connections share this endpoint per browser tab:
//   - Datastar @get (data-init in playgame.templ): sends ?datastar=… query param,
//     only expects Datastar-format events (event: datastar-patch-*).
//   - Native EventSource (initClock.js): no query param, only expects plain JSON
//     events (data: {"type":"clock_tick",…}).
//
// Mixing both event formats on the same connection causes Datastar's SSE parser
// to throw "Error in input stream", drop the connection, and miss board updates.
// We detect which client type each connection is and filter accordingly.
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
	// the clockUnlocked signal is broadcast — the second player to connect triggers it.
	ch := g.Hub.Subscribe()
	defer g.Hub.Unsubscribe(ch)

	bothJustConnected := g.PlayMeta.RecordSSEConnect(role)
	defer func() {
		g.PlayMeta.RecordSSEDisconnect(role)
		whiteOnline, blackOnline := g.PlayMeta.OnlineStatus()
		g.Hub.BroadcastSignals(map[string]any{
			"whiteOnline": whiteOnline, "blackOnline": blackOnline,
		})
	}()

	if bothJustConnected && g.IsOngoing() {
		deadline := time.Now().Add(20 * time.Second)
		g.StartPlayClock(deadline)
		g.Hub.BroadcastSignals(map[string]any{
			"clockUnlocked":       true,
			"firstMoveDeadlineNs": deadline.UnixNano(),
		})
	}

	// Initial state for this connection: board snapshot + current clock + any
	// pending rematch + current online status. Written directly so the new
	// client gets correct values without waiting for a hub broadcast.
	syncDatastarBoard(ctx, c, g)
	if g.Timed {
		writeClockSnapshot(c, g.ClockTickSnapshot())
	}
	whiteOnline, blackOnline := g.PlayMeta.OnlineStatus()
	initialSignals := map[string]any{
		"whiteOnline": whiteOnline, "blackOnline": blackOnline,
	}
	if dlNs := g.PlayMeta.GetFirstMoveDeadlineNs(); dlNs != 0 && len(g.History) == 0 && g.IsOngoing() {
		initialSignals["firstMoveDeadlineNs"] = dlNs
	}
	writeSignalsDirect(c, initialSignals)
	if proposer := g.PlayMeta.GetRematchProposer(); proposer != engine.NoColor {
		proposerRole := "white"
		if proposer == engine.Black {
			proposerRole = "black"
		}
		writeSignalsDirect(c, map[string]any{"rematchProposedBy": proposerRole})
	}

	// Seed keepaliveTs immediately so the client watchdog has a fresh baseline.
	writeSignalsDirect(c, map[string]any{"keepaliveTs": time.Now().UnixNano()})

	// Keepalive cadence: a comment line every 5s keeps proxies (Cloudflare,
	// nginx, ALB) from idle-closing the SSE stream. Every other tick we also
	// push a keepaliveTs signal patch (10s cadence) so the client's
	// reconnect watchdog in initClock.js can detect a dead stream without
	// needing a move to happen.
	keepaliveTicker := time.NewTicker(5 * time.Second)
	defer keepaliveTicker.Stop()
	keepaliveTickCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-keepaliveTicker.C:
			c.Writer.WriteString(":\n\n")
			c.Writer.Flush()
			keepaliveTickCount++
			if keepaliveTickCount%2 == 0 {
				writeSignalsDirect(c, map[string]any{"keepaliveTs": time.Now().UnixNano()})
			}
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

	g.Hub.BroadcastSignals(map[string]any{"rematchProposedBy": claims.Role})

	// The watchdog exits when the game ends, so expire the rematch from a dedicated goroutine.
	go func() {
		time.Sleep(30 * time.Second)
		if g.PlayMeta.ClearRematchIfPendingBy(proposerColor) {
			g.Hub.BroadcastSignals(map[string]any{"rematchProposedBy": ""})
		}
	}()

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

	g.Hub.BroadcastSignals(map[string]any{"rematchAcceptedUrl": proposerURL})

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
	g.Hub.BroadcastSignals(map[string]any{"rematchProposedBy": ""})
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
		g.Hub.BroadcastSignals(map[string]any{
			"gameState":     int(game.GameAbandoned),
			"gameStateText": "Game cancelled",
		})
	}

	_ = ctx
	ClearPlaySession(c)
	c.Redirect(http.StatusSeeOther, "/")
}

var errNotPlayGame = errors.New("not a play game")

// loadPlayGameFromDB restores an online play game from the database after a server restart.
func loadPlayGameFromDB(ctx context.Context, dbRepo db.Repository, gameID string) (*game.Game, error) {
	core, err := loadGameCore(ctx, dbRepo, gameID)
	if err != nil {
		return nil, err
	}
	if core.dbGame.WhiteParticipantID == nil || core.dbGame.BlackParticipantID == nil {
		return nil, errNotPlayGame
	}

	participants, err := dbRepo.Participants().ListBySession(ctx, core.dbGame.SessionID)
	if err != nil {
		return nil, err
	}

	var timeCtrlNs, restoredIncNs int64
	if core.dbGame.TimeControlNs != nil {
		timeCtrlNs = *core.dbGame.TimeControlNs
	}
	if core.dbGame.IncrementNs != nil {
		restoredIncNs = *core.dbGame.IncrementNs
	}

	meta := &game.PlayMeta{
		SessionID:            core.dbGame.SessionID,
		OriginalMode:         game.GameMode{TimeNs: timeCtrlNs, Increment: restoredIncNs, Variant: core.dbGame.Variant, Timed: timeCtrlNs > 0},
		RematchProposedBy:    engine.NoColor,
		ClaimVictoryFor:      engine.NoColor,
		BothPlayersConnectedOnce: core.state != game.GameOngoing,
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

	restored := game.NewRestoredPlayGame(gameID, core.board, core.clock, core.history, core.seq, core.state, core.winner, meta)

	if core.state == game.GameOngoing {
		initGameBatch(restored, core.dbGame.SessionID, dbRepo)
	}

	return restored, nil
}

