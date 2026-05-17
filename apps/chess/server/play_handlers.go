package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
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
)

const (
	// joinTimeout is how long after game creation the opponent has to claim
	// their seat before the game auto-cancels.
	joinTimeout = 120 * time.Second

	// firstMoveTimeout is how long white has to play the first move after
	// both players have connected.
	firstMoveTimeout = 60 * time.Second
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

	joinDeadline := time.Now().Add(joinTimeout)
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
		DrawOfferedBy:      engine.NoColor,
		TakebackOfferedBy:  engine.NoColor,
		LastDrawSeq:        [2]int64{-1, -1},
		LastTakebackSeq:    [2]int64{-1, -1},
		ClaimVictoryFor:    engine.NoColor,
		JoinDeadline:       &joinDeadline,
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

	// If the visitor has a valid cookie for this game, render the game directly.
	if claims, err := GetPlaySession(c.Request); err == nil && claims.GameID == g.ID {
		role := claims.Role
		flipped := role == "black"
		Render(c, http.StatusOK, pages.PlayGamePage(g, role, flipped, CSRFToken(c)))
		return
	}

	// Visitor has no session cookie for this game.
	//
	// If a ?token= is present, surface a join-confirmation page rather than
	// claiming the seat on GET. Claiming on GET allows link-preview crawlers
	// (WhatsApp, Slack, etc.) to consume the token when the URL is pasted
	// into a chat — the real opponent then arrives to find the seat already
	// claimed and is silently demoted to spectator.
	if token := c.Query("token"); token != "" {
		// Bots get a minimal OG-preview page; they must not consume the token
		// or even learn about the seat assignment.
		if isLinkPreviewBot(c.Request.UserAgent()) {
			Render(c, http.StatusOK, pages.PlayInvitePreview(g))
			return
		}

		// Determine which side this token corresponds to (without claiming).
		intendedRole := ""
		switch token {
		case g.PlayMeta.WhiteToken:
			intendedRole = "white"
		case g.PlayMeta.BlackToken:
			intendedRole = "black"
		}
		if intendedRole == "" {
			c.Redirect(http.StatusSeeOther, "/?error=invalid_invite")
			return
		}
		// If that seat is already claimed (e.g. opponent on another device),
		// fall through to spectator.
		claimed := g.PlayMeta.IsSeatClaimed(intendedRole)
		if claimed {
			Render(c, http.StatusOK, pages.PlayGamePage(g, "spectator", false, CSRFToken(c)))
			return
		}
		Render(c, http.StatusOK, pages.PlayJoinConfirm(g, intendedRole, token, CSRFToken(c)))
		return
	}

	// No cookie, no token → spectator view of the game.
	Render(c, http.StatusOK, pages.PlayGamePage(g, "spectator", false, CSRFToken(c)))
}

// ClaimPlaySeat handles POST /play/:gameID/claim — completes the join after
// the visitor has explicitly confirmed on the join page. The token must be
// supplied via form field "token".
func ClaimPlaySeat(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/?error=server_error")
		return
	}
	gameID := c.Param("gameID")
	g, ok := repo.Get(gameID)
	if !ok || g.PlayMeta == nil {
		c.Redirect(http.StatusSeeOther, "/?error=game_not_found")
		return
	}
	token := c.PostForm("token")
	if token == "" {
		c.Redirect(http.StatusSeeOther, "/?error=invalid_invite")
		return
	}
	// If the caller already has a session for this game, just send them to the
	// board — no further claim needed.
	if claims, err := GetPlaySession(c.Request); err == nil && claims.GameID == gameID {
		c.Redirect(http.StatusSeeOther, "/play/"+gameID)
		return
	}
	participantID, claimedRole, ok := g.PlayMeta.TryClaim(token)
	if !ok {
		// Token unknown or seat already taken — drop to spectator view.
		c.Redirect(http.StatusSeeOther, "/play/"+gameID)
		return
	}
	claimParticipant(c, g, participantID, claimedRole)
	c.Redirect(http.StatusSeeOther, "/play/"+gameID)
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

	// Clear the per-request write deadline — the base server's WriteTimeout
	// would otherwise kill this long-lived SSE stream at 60s.
	disableSSETimeouts(c)

	// Brotli-encode the stream when the client supports it. Frequent flushes
	// (every keepalive + every broadcast frame) emit brotli sync flushes so
	// bytes reach the browser without extra latency.
	defer maybeBrotliSSE(c)()

	// Subscribe before RecordSSEConnect so this client's channel is present when
	// the clockUnlocked signal is broadcast — the second player to connect triggers it.
	ch := g.Hub.Subscribe()
	defer g.Hub.Unsubscribe(ch)

	bothFirstConnected, shouldArmClock := g.PlayMeta.RecordSSEConnect(role)
	// Broadcast updated online status so the opponent's UI flips the dot
	// back to green and clears the forfeit countdown.
	{
		whiteOnline, blackOnline := g.PlayMeta.OnlineStatus()
		wDisAt, bDisAt := g.PlayMeta.DisconnectAtNs()
		go g.Hub.BroadcastSignals(map[string]any{
			"whiteOnline":         whiteOnline,
			"blackOnline":         blackOnline,
			"whiteDisconnectAtNs": wDisAt,
			"blackDisconnectAtNs": bDisAt,
		})
	}
	defer func() {
		g.PlayMeta.RecordSSEDisconnect(role)
		whiteOnline, blackOnline := g.PlayMeta.OnlineStatus()
		wDisAt, bDisAt := g.PlayMeta.DisconnectAtNs()
		g.Hub.BroadcastSignals(map[string]any{
			"whiteOnline":         whiteOnline,
			"blackOnline":         blackOnline,
			"whiteDisconnectAtNs": wDisAt,
			"blackDisconnectAtNs": bDisAt,
		})
	}()

	if g.IsOngoing() {
		switch {
		case bothFirstConnected:
			// Brand-new game, both players just arrived for the first time.
			// Start the 20s first-move countdown for white.
			deadline := time.Now().Add(firstMoveTimeout)
			g.StartPlayClock(deadline)
			g.Hub.BroadcastSignals(map[string]any{
				"clockUnlocked":       true,
				"firstMoveDeadlineNs": deadline.UnixNano(),
				"joinDeadlineNs":      int64(0),
			})
		case shouldArmClock:
			// Game already had at least one move (or BothPlayersConnectedOnce
			// was preserved from a prior session). Resume the side-to-move
			// clock from its persisted remaining time — downtime is not
			// billed to either player.
			g.ResumeClockForActiveSide()
		}
	}

	// Initial state for this connection: board snapshot + current clock + any
	// pending rematch + current online status. Written directly so the new
	// client gets correct values without waiting for a hub broadcast.
	syncDatastarBoard(ctx, c, g, role)
	if g.Timed {
		writeClockSnapshot(c, g.ClockTickSnapshot())
	}
	whiteOnline, blackOnline := g.PlayMeta.OnlineStatus()
	wDisAtInit, bDisAtInit := g.PlayMeta.DisconnectAtNs()
	initialSignals := map[string]any{
		"whiteOnline":         whiteOnline,
		"blackOnline":         blackOnline,
		"whiteDisconnectAtNs": wDisAtInit,
		"blackDisconnectAtNs": bDisAtInit,
	}
	if dlNs := g.PlayMeta.GetFirstMoveDeadlineNs(); dlNs != 0 && len(g.History) == 0 && g.IsOngoing() {
		initialSignals["firstMoveDeadlineNs"] = dlNs
	}
	if jdNs := g.PlayMeta.GetJoinDeadlineNs(); jdNs != 0 && g.IsOngoing() {
		initialSignals["joinDeadlineNs"] = jdNs
	} else {
		initialSignals["joinDeadlineNs"] = int64(0)
	}
	writeSignalsDirect(c, initialSignals)
	if proposer := g.PlayMeta.GetRematchProposer(); proposer != engine.NoColor {
		proposerRole := "white"
		if proposer == engine.Black {
			proposerRole = "black"
		}
		writeSignalsDirect(c, map[string]any{"rematchProposedBy": proposerRole})
	}
	if offerer := g.PlayMeta.GetDrawOfferer(); offerer != engine.NoColor {
		offererRole := "white"
		if offerer == engine.Black {
			offererRole = "black"
		}
		writeSignalsDirect(c, map[string]any{"drawOfferedBy": offererRole})
	}
	if proposer := g.PlayMeta.GetTakebackProposer(); proposer != engine.NoColor {
		proposerRole := "white"
		if proposer == engine.Black {
			proposerRole = "black"
		}
		writeSignalsDirect(c, map[string]any{"takebackOfferedBy": proposerRole})
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

// PlayMove handles POST /play/:gameID/move/:from/:to[?promo=N].
func PlayMove(c *gin.Context) {
	ctx := c.Request.Context()
	repo, ok := store.GetRepoFromContext(ctx)
	if !ok {
		return
	}

	gameID, _ := c.Params.Get("gameID")
	from, to, promo, err := parseMoveParams(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

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

	prevHistoryLen := g.HistoryLen()
	applyMoveRequest(c, g, from, to, promo)
	if g.HistoryLen() > prevHistoryLen {
		// A move landed: any pending mid-game offer is implicitly declined.
		cleared := map[string]any{}
		if g.PlayMeta.GetDrawOfferer() != engine.NoColor {
			g.PlayMeta.ClearDraw()
			cleared["drawOfferedBy"] = ""
		}
		if g.PlayMeta.GetTakebackProposer() != engine.NoColor {
			g.PlayMeta.ClearTakeback()
			cleared["takebackOfferedBy"] = ""
		}
		if len(cleared) > 0 {
			g.Hub.BroadcastSignals(cleared)
		}
	}
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

// ClaimDraw handles POST /play/:gameID/claim-draw. Currently the only
// player-claimable draw is the 50-move rule (threefold and 75-move both
// auto-trigger in UpdateGameState). Returns 409 if the claim is no longer
// valid — typically because a pawn move or capture reset the halfmove clock
// between the client deciding to claim and the request landing.
func ClaimDraw(c *gin.Context) {
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

	if !g.ClaimFiftyMoveDraw() {
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
		DrawOfferedBy:      engine.NoColor,
		TakebackOfferedBy:  engine.NoColor,
		LastDrawSeq:        [2]int64{-1, -1},
		LastTakebackSeq:    [2]int64{-1, -1},
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

// ResignGame handles POST /play/:gameID/resign.
func ResignGame(c *gin.Context) {
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

	loser := engine.White
	if claims.Role == "black" {
		loser = engine.Black
	}
	if !g.Resign(loser) {
		c.Status(http.StatusConflict)
		return
	}
	// Resign also voids any pending offers so prompts close.
	g.PlayMeta.ClearRematch()
	g.PlayMeta.ClearDraw()
	g.PlayMeta.ClearTakeback()
	g.Hub.BroadcastSignals(map[string]any{
		"drawOfferedBy":     "",
		"takebackOfferedBy": "",
	})
	_ = ctx
	c.Status(http.StatusNoContent)
}

// ProposeDraw handles POST /play/:gameID/draw.
func ProposeDraw(c *gin.Context) {
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
	if !g.IsOngoing() {
		c.Status(http.StatusConflict)
		return
	}

	proposerColor := engine.White
	if claims.Role == "black" {
		proposerColor = engine.Black
	}
	if !g.PlayMeta.TryProposeDraw(proposerColor, g.Seq) {
		c.Status(http.StatusConflict)
		return
	}
	lockKey := "lastDrawSeqWhite"
	if proposerColor == engine.Black {
		lockKey = "lastDrawSeqBlack"
	}
	g.Hub.BroadcastSignals(map[string]any{
		"drawOfferedBy": claims.Role,
		lockKey:         int64(g.Seq),
	})

	go func() {
		time.Sleep(30 * time.Second)
		if g.PlayMeta.ClearDrawIfPendingBy(proposerColor) {
			g.Hub.BroadcastSignals(map[string]any{"drawOfferedBy": ""})
		}
	}()

	_ = ctx
	c.Status(http.StatusNoContent)
}

// AcceptDraw handles POST /play/:gameID/draw/accept.
func AcceptDraw(c *gin.Context) {
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
	if _, ok := g.PlayMeta.AcceptAndClearDraw(accepterColor); !ok {
		c.Status(http.StatusConflict)
		return
	}
	if !g.AgreeDraw() {
		c.Status(http.StatusConflict)
		return
	}
	g.Hub.BroadcastSignals(map[string]any{"drawOfferedBy": ""})
	_ = ctx
	c.Status(http.StatusNoContent)
}

// DeclineDraw handles POST /play/:gameID/draw/decline.
func DeclineDraw(c *gin.Context) {
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

	g.PlayMeta.ClearDraw()
	g.Hub.BroadcastSignals(map[string]any{"drawOfferedBy": ""})
	_ = ctx
	c.Status(http.StatusNoContent)
}

// ProposeTakeback handles POST /play/:gameID/takeback.
// Only proposable when it's the opponent's turn (i.e. the requester just moved
// and the opponent has not yet replied).
func ProposeTakeback(c *gin.Context) {
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
	if !g.IsOngoing() || g.HistoryLen() == 0 {
		c.Status(http.StatusConflict)
		return
	}

	proposerColor := engine.White
	if claims.Role == "black" {
		proposerColor = engine.Black
	}
	// Requester must have just moved → board's side-to-move is the opponent.
	if g.Board.SideToMove == proposerColor {
		c.Status(http.StatusConflict)
		return
	}
	if !g.PlayMeta.TryProposeTakeback(proposerColor, g.Seq) {
		c.Status(http.StatusConflict)
		return
	}
	lockKey := "lastTakebackSeqWhite"
	if proposerColor == engine.Black {
		lockKey = "lastTakebackSeqBlack"
	}
	g.Hub.BroadcastSignals(map[string]any{
		"takebackOfferedBy": claims.Role,
		lockKey:             int64(g.Seq),
	})

	go func() {
		time.Sleep(30 * time.Second)
		if g.PlayMeta.ClearTakebackIfPendingBy(proposerColor) {
			g.Hub.BroadcastSignals(map[string]any{"takebackOfferedBy": ""})
		}
	}()

	_ = ctx
	c.Status(http.StatusNoContent)
}

// AcceptTakeback handles POST /play/:gameID/takeback/accept.
// Reverts the requester's last move (1 ply). Clocks are not refunded.
func AcceptTakeback(c *gin.Context) {
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
	if _, ok := g.PlayMeta.AcceptAndClearTakeback(accepterColor); !ok {
		c.Status(http.StatusConflict)
		return
	}
	if !g.RevertLastPly() {
		c.Status(http.StatusConflict)
		return
	}
	// A takeback invalidates any in-flight draw offer too — the position has
	// changed materially.
	clearedDraw := false
	if g.PlayMeta.GetDrawOfferer() != engine.NoColor {
		g.PlayMeta.ClearDraw()
		clearedDraw = true
	}

	signals := ui_store.ChessBoardSignalsFromGame(g)
	hubBroadcastBoard(ctx, g, signals)
	extras := map[string]any{"takebackOfferedBy": ""}
	if clearedDraw {
		extras["drawOfferedBy"] = ""
	}
	if g.Timed {
		nowNs := time.Now().UnixNano()
		extras["clkW"] = g.Clock.RemainingAt(0, nowNs)
		extras["clkB"] = g.Clock.RemainingAt(1, nowNs)
		extras["clkWRun"] = g.Clock.White.Running
		extras["clkBRun"] = g.Clock.Black.Running
		extras["clkTs"] = nowNs
	}
	g.Hub.BroadcastSignals(extras)
	c.Status(http.StatusNoContent)
}

// DeclineTakeback handles POST /play/:gameID/takeback/decline.
func DeclineTakeback(c *gin.Context) {
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

	g.PlayMeta.ClearTakeback()
	g.Hub.BroadcastSignals(map[string]any{"takebackOfferedBy": ""})
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

	// BothPlayersConnectedOnce is true if the game already has moves (it must
	// have been "started" before — both players were online to allow the first
	// move) or if it's finished (gating doesn't matter). For an ongoing game
	// with no moves yet, leave it false so the first-move flow runs when both
	// players reconnect.
	bothConnectedBefore := core.state != game.GameOngoing || len(core.history) > 0
	meta := &game.PlayMeta{
		SessionID:                core.dbGame.SessionID,
		OriginalMode:             game.GameMode{TimeNs: timeCtrlNs, Increment: restoredIncNs, Variant: core.dbGame.Variant, Timed: timeCtrlNs > 0},
		RematchProposedBy:        engine.NoColor,
		DrawOfferedBy:            engine.NoColor,
		TakebackOfferedBy:        engine.NoColor,
		LastDrawSeq:              [2]int64{-1, -1},
		LastTakebackSeq:          [2]int64{-1, -1},
		ClaimVictoryFor:          engine.NoColor,
		BothPlayersConnectedOnce: bothConnectedBefore,
		// ClockArmedAfterLoad stays false so the first reconnect-pair after
		// the restart re-starts the side-to-move clock from its saved value.
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

