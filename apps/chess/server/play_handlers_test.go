package server_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/server"
	"github.com/lordsonvimal/synergy/apps/chess/store"
)

// newPlayTestRouter builds a router and exposes the in-memory game store so
// tests can peek at PlayMeta state directly.
func newPlayTestRouter(t *testing.T) (*gin.Engine, *store.GameStore) {
	t.Helper()
	t.Setenv("DATA_DIR", t.TempDir())
	gin.SetMode(gin.TestMode)

	router := gin.New()
	gs := store.NewGameStore()

	router.Use(store.StoreContext(gs))
	router.Use(server.CSRFMiddleware())
	server.InitRoutes(router)
	return router, gs
}

// createPlayGame creates a /play game via POST and returns the game ID and
// the creator's play_session cookie.
func createPlayGame(t *testing.T, r *gin.Engine) (gameID string, sessionCookie *http.Cookie) {
	t.Helper()
	req := postWithCSRF("/play", url.Values{"mode": {"Blitz 5+2"}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("POST /play status: got %d, want %d (body=%q)", w.Code, http.StatusSeeOther, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "/play/") {
		t.Fatalf("expected /play/<id> redirect, got %q", loc)
	}
	gameID = strings.TrimPrefix(loc, "/play/")
	for _, c := range w.Result().Cookies() {
		if c.Name == "play_session" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatalf("CreatePlay did not set a play_session cookie")
	}
	return gameID, sessionCookie
}

func getPlayGame(t *testing.T, gs *store.GameStore, id string) *game.Game {
	t.Helper()
	g, ok := gs.Get(id)
	if !ok {
		t.Fatalf("game %q not found in store", id)
	}
	if g.PlayMeta == nil {
		t.Fatalf("game %q has no PlayMeta (not a play game)", id)
	}
	return g
}

// ---- CreatePlay ----------------------------------------------------------

func TestCreatePlayRedirectsToPlayRoute(t *testing.T) {
	r, _ := newPlayTestRouter(t)
	gameID, _ := createPlayGame(t, r)
	if gameID == "" {
		t.Fatal("expected non-empty game ID")
	}
}

func TestCreatePlayInvalidModeRedirectsWithError(t *testing.T) {
	r, _ := newPlayTestRouter(t)
	req := postWithCSRF("/play", url.Values{"mode": {"nonsense"}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Header().Get("Location"), "invalid_mode") {
		t.Errorf("expected redirect with invalid_mode, got %q", w.Header().Get("Location"))
	}
}

func TestCreatePlaySetsJoinDeadlineAndClaimsCreatorSeat(t *testing.T) {
	r, gs := newPlayTestRouter(t)
	gameID, cookie := createPlayGame(t, r)
	g := getPlayGame(t, gs, gameID)

	if dl := g.PlayMeta.GetJoinDeadlineNs(); dl == 0 {
		t.Fatal("expected JoinDeadline to be set on create")
	} else {
		// Should be ~120s in the future (within a generous window).
		rem := time.Duration(dl-time.Now().UnixNano()) * time.Nanosecond
		if rem < 100*time.Second || rem > 121*time.Second {
			t.Errorf("JoinDeadline should be ~120s out, got %v", rem)
		}
	}

	// Exactly one seat is claimed by the creator; the other is open.
	wc := g.PlayMeta.IsSeatClaimed("white")
	bc := g.PlayMeta.IsSeatClaimed("black")
	if wc == bc {
		t.Errorf("exactly one seat should be claimed on create, got white=%v black=%v", wc, bc)
	}

	// Creator cookie role matches the claimed seat.
	role := decodeRole(t, cookie.Value)
	if role == "white" && !wc {
		t.Error("cookie says white but white seat not claimed")
	}
	if role == "black" && !bc {
		t.Error("cookie says black but black seat not claimed")
	}
}

func TestCreatePlayBlocksWhenAlreadyInGame(t *testing.T) {
	r, _ := newPlayTestRouter(t)
	_, cookie := createPlayGame(t, r)

	req := postWithCSRF("/play", url.Values{"mode": {"Blitz 5+2"}})
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Header().Get("Location"), "already_in_game") {
		t.Errorf("expected redirect with already_in_game, got %q", w.Header().Get("Location"))
	}
}

// ---- ShowPlayGame: cookie / no-token paths -------------------------------

func TestShowPlayGameCreatorSeesGameBoard(t *testing.T) {
	r, _ := newPlayTestRouter(t)
	gameID, cookie := createPlayGame(t, r)

	req := httptest.NewRequest(http.MethodGet, "/play/"+gameID, nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /play/:id status: got %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `data-testid="chessboard"`) {
		t.Error("expected chessboard on creator's game page")
	}
	if !strings.Contains(body, "Waiting for your opponent") {
		t.Error("expected share-links panel on creator's page while opponent has not joined")
	}
}

func TestShowPlayGameNoCookieNoTokenIsSpectator(t *testing.T) {
	r, _ := newPlayTestRouter(t)
	gameID, _ := createPlayGame(t, r)

	req := httptest.NewRequest(http.MethodGet, "/play/"+gameID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	body := w.Body.String()
	if !containsRoleSignal(body, "spectator") {
		t.Errorf("expected spectator role in signals payload, body did not contain it")
	}
	// Spectators must NOT see the invite share links.
	if strings.Contains(body, "Waiting for your opponent") {
		t.Error("spectator should not see the invite share-links panel")
	}
}

func TestShowPlayGameMissingRedirects(t *testing.T) {
	r, _ := newPlayTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/play/does-not-exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Errorf("missing game GET: got %d, want redirect", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "game_not_found") {
		t.Errorf("expected game_not_found redirect, got %q", w.Header().Get("Location"))
	}
}

// ---- ShowPlayGame: ?token= flow (the bug-fix surface) --------------------

func TestShowPlayGameWithTokenAsBotServesPreviewAndDoesNotClaim(t *testing.T) {
	r, gs := newPlayTestRouter(t)
	gameID, _ := createPlayGame(t, r)
	g := getPlayGame(t, gs, gameID)

	openToken, openRole := openSeat(g)
	preClaimed := g.PlayMeta.IsSeatClaimed(openRole)

	req := httptest.NewRequest(http.MethodGet, "/play/"+gameID+"?token="+openToken, nil)
	req.Header.Set("User-Agent", "WhatsApp/2.23.24.84 A")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("bot fetch: got %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<meta property=\"og:title\"") {
		t.Error("expected OpenGraph meta tags in bot preview")
	}
	if strings.Contains(body, `data-testid="chessboard"`) {
		t.Error("bot preview must NOT include the game board UI")
	}
	if strings.Contains(body, "Join as") {
		t.Error("bot preview must NOT include the join-confirmation form")
	}

	// Critically: the seat must remain unclaimed.
	if g.PlayMeta.IsSeatClaimed(openRole) != preClaimed {
		t.Errorf("bot fetch must not claim the seat (pre=%v, post=%v)", preClaimed, g.PlayMeta.IsSeatClaimed(openRole))
	}
}

func TestShowPlayGameWithTokenAsRealUserShowsJoinConfirm(t *testing.T) {
	r, gs := newPlayTestRouter(t)
	gameID, _ := createPlayGame(t, r)
	g := getPlayGame(t, gs, gameID)
	openToken, openRole := openSeat(g)

	req := httptest.NewRequest(http.MethodGet, "/play/"+gameID+"?token="+openToken, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 Version/17.2 Mobile/15E148 Safari/604.1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("real-user GET ?token=: got %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Join as ") {
		t.Error("expected join-confirmation page with 'Join as ...' button")
	}
	if !strings.Contains(body, `action="/play/`+gameID+`/claim"`) {
		t.Error("expected confirmation form posting to /play/:id/claim")
	}
	if !strings.Contains(body, openToken) {
		t.Error("expected confirmation form to include the token as a hidden field")
	}
	// Seat must remain unclaimed until the user POSTs.
	if g.PlayMeta.IsSeatClaimed(openRole) {
		t.Errorf("GET with token must not claim the %s seat", openRole)
	}
}

func TestShowPlayGameWithUnknownTokenRedirectsWithError(t *testing.T) {
	r, _ := newPlayTestRouter(t)
	gameID, _ := createPlayGame(t, r)

	req := httptest.NewRequest(http.MethodGet, "/play/"+gameID+"?token=deadbeef", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("unknown token: got %d, want redirect", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "invalid_invite") {
		t.Errorf("expected invalid_invite redirect, got %q", w.Header().Get("Location"))
	}
}

func TestShowPlayGameWithAlreadyClaimedTokenFallsThroughToSpectator(t *testing.T) {
	r, gs := newPlayTestRouter(t)
	gameID, _ := createPlayGame(t, r)
	g := getPlayGame(t, gs, gameID)
	openToken, openRole := openSeat(g)

	// First claim succeeds.
	if _, _, ok := g.PlayMeta.TryClaim(openToken); !ok {
		t.Fatal("first TryClaim should succeed")
	}
	if !g.PlayMeta.IsSeatClaimed(openRole) {
		t.Fatalf("expected %s seat claimed after TryClaim", openRole)
	}

	// Second visitor (no cookie) lands on the URL → spectator, no error.
	req := httptest.NewRequest(http.MethodGet, "/play/"+gameID+"?token="+openToken, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("post-claim GET: got %d, want 200", w.Code)
	}
	if !containsRoleSignal(w.Body.String(), "spectator") {
		t.Error("post-claim visitor should see spectator role")
	}
}

func TestShowPlayGameWithCookieIgnoresQueryToken(t *testing.T) {
	r, gs := newPlayTestRouter(t)
	gameID, creatorCookie := createPlayGame(t, r)
	g := getPlayGame(t, gs, gameID)
	openToken, openRole := openSeat(g)
	preClaimed := g.PlayMeta.IsSeatClaimed(openRole)

	// Creator visits the URL with the *opponent's* token query param attached.
	// They already have a session — TryClaim must NOT be invoked for them.
	req := httptest.NewRequest(http.MethodGet, "/play/"+gameID+"?token="+openToken, nil)
	req.AddCookie(creatorCookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	if g.PlayMeta.IsSeatClaimed(openRole) != preClaimed {
		t.Errorf("creator-with-cookie must not consume opponent token (pre=%v, post=%v)",
			preClaimed, g.PlayMeta.IsSeatClaimed(openRole))
	}
}

// ---- ClaimPlaySeat (POST) ------------------------------------------------

func TestClaimPlaySeatSucceeds(t *testing.T) {
	r, gs := newPlayTestRouter(t)
	gameID, _ := createPlayGame(t, r)
	g := getPlayGame(t, gs, gameID)
	openToken, openRole := openSeat(g)

	req := postWithCSRF("/play/"+gameID+"/claim", url.Values{"token": {openToken}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("POST /play/:id/claim: got %d (body=%q), want redirect", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Location"); got != "/play/"+gameID {
		t.Errorf("expected redirect to /play/%s, got %q", gameID, got)
	}
	if !g.PlayMeta.IsSeatClaimed(openRole) {
		t.Errorf("%s seat should be claimed after POST", openRole)
	}

	var session *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "play_session" {
			session = c
		}
	}
	if session == nil {
		t.Fatal("claim should set a play_session cookie for the joiner")
	}
	if role := decodeRole(t, session.Value); role != openRole {
		t.Errorf("joiner cookie role = %q, want %q", role, openRole)
	}
}

func TestClaimPlaySeatMissingTokenRedirectsWithError(t *testing.T) {
	r, _ := newPlayTestRouter(t)
	gameID, _ := createPlayGame(t, r)

	req := postWithCSRF("/play/"+gameID+"/claim", url.Values{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Header().Get("Location"), "invalid_invite") {
		t.Errorf("expected invalid_invite redirect, got %q", w.Header().Get("Location"))
	}
}

func TestClaimPlaySeatUnknownTokenDropsToSpectator(t *testing.T) {
	r, gs := newPlayTestRouter(t)
	gameID, _ := createPlayGame(t, r)
	g := getPlayGame(t, gs, gameID)
	_, openRole := openSeat(g)
	preClaimed := g.PlayMeta.IsSeatClaimed(openRole)

	req := postWithCSRF("/play/"+gameID+"/claim", url.Values{"token": {"unknown-token"}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want redirect", w.Code)
	}
	if w.Header().Get("Location") != "/play/"+gameID {
		t.Errorf("expected redirect to /play/%s, got %q", gameID, w.Header().Get("Location"))
	}
	if g.PlayMeta.IsSeatClaimed(openRole) != preClaimed {
		t.Errorf("unknown token must not claim a seat")
	}
	// No session cookie should be set.
	for _, c := range w.Result().Cookies() {
		if c.Name == "play_session" {
			t.Errorf("unknown token must not set a play_session cookie, got %+v", c)
		}
	}
}

func TestClaimPlaySeatRejectsDoubleClaim(t *testing.T) {
	r, gs := newPlayTestRouter(t)
	gameID, _ := createPlayGame(t, r)
	g := getPlayGame(t, gs, gameID)
	openToken, openRole := openSeat(g)

	// First claim from real user A — sets cookie.
	req1 := postWithCSRF("/play/"+gameID+"/claim", url.Values{"token": {openToken}})
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if !g.PlayMeta.IsSeatClaimed(openRole) {
		t.Fatalf("first claim should succeed")
	}

	// Second POST with same token from a *different* visitor (no cookie).
	// Must not set a cookie and must not re-claim.
	req2 := postWithCSRF("/play/"+gameID+"/claim", url.Values{"token": {openToken}})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	for _, c := range w2.Result().Cookies() {
		if c.Name == "play_session" {
			t.Errorf("second claim attempt must not set a play_session cookie, got %+v", c)
		}
	}
}

func TestClaimPlaySeatWithExistingSessionJustRedirects(t *testing.T) {
	r, gs := newPlayTestRouter(t)
	gameID, creatorCookie := createPlayGame(t, r)
	g := getPlayGame(t, gs, gameID)
	openToken, openRole := openSeat(g)

	// Creator (already has session) POSTs the opponent's token — must be a no-op claim.
	req := postWithCSRF("/play/"+gameID+"/claim", url.Values{"token": {openToken}})
	req.AddCookie(creatorCookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want redirect", w.Code)
	}
	if g.PlayMeta.IsSeatClaimed(openRole) {
		t.Errorf("creator's POST must not claim the opponent's seat")
	}
}

func TestClaimPlaySeatMissingGameRedirects(t *testing.T) {
	r, _ := newPlayTestRouter(t)
	req := postWithCSRF("/play/does-not-exist/claim", url.Values{"token": {"x"}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Header().Get("Location"), "game_not_found") {
		t.Errorf("expected game_not_found redirect, got %q", w.Header().Get("Location"))
	}
}

// ---- End-to-end happy path -----------------------------------------------

// TestEndToEnd_WhatsAppPreviewThenRealOpponent reproduces the original bug:
// a WhatsApp link-preview fetch must NOT consume the token, so the real
// opponent (clicking the link from the chat) can still join correctly.
func TestEndToEnd_WhatsAppPreviewThenRealOpponent(t *testing.T) {
	r, gs := newPlayTestRouter(t)
	gameID, creatorCookie := createPlayGame(t, r)
	g := getPlayGame(t, gs, gameID)
	openToken, openRole := openSeat(g)
	creatorRole := otherRole(openRole)

	// (1) Creator pastes the invite link in WhatsApp → WhatsApp's bot crawls
	// the URL to render a preview. Must NOT claim.
	bot := httptest.NewRequest(http.MethodGet, "/play/"+gameID+"?token="+openToken, nil)
	bot.Header.Set("User-Agent", "WhatsApp/2.23.24.84 A")
	wBot := httptest.NewRecorder()
	r.ServeHTTP(wBot, bot)
	if g.PlayMeta.IsSeatClaimed(openRole) {
		t.Fatal("regression: WhatsApp preview consumed the token")
	}

	// (2) Real opponent opens the link → join confirmation page.
	confirm := httptest.NewRequest(http.MethodGet, "/play/"+gameID+"?token="+openToken, nil)
	confirm.Header.Set("User-Agent", "Mozilla/5.0 (iPhone) Safari/605")
	wConfirm := httptest.NewRecorder()
	r.ServeHTTP(wConfirm, confirm)
	if wConfirm.Code != http.StatusOK || !strings.Contains(wConfirm.Body.String(), "Join as ") {
		t.Fatalf("real opponent should see join-confirm page, got status=%d", wConfirm.Code)
	}
	if g.PlayMeta.IsSeatClaimed(openRole) {
		t.Fatal("GET with token must not claim — only POST claim does")
	}

	// (3) Opponent confirms → POST /play/:id/claim → seat is claimed.
	post := postWithCSRF("/play/"+gameID+"/claim", url.Values{"token": {openToken}})
	wPost := httptest.NewRecorder()
	r.ServeHTTP(wPost, post)
	if wPost.Code != http.StatusSeeOther {
		t.Fatalf("claim POST: got %d, want redirect", wPost.Code)
	}
	if !g.PlayMeta.IsSeatClaimed(openRole) {
		t.Fatal("opponent's POST should claim the seat")
	}

	// (4) Both seats now claimed; the creator's cookie role and the
	// opponent's cookie role must be opposites — so each player gets the
	// piece they were assigned at creation time.
	var opponentCookie *http.Cookie
	for _, c := range wPost.Result().Cookies() {
		if c.Name == "play_session" {
			opponentCookie = c
		}
	}
	if opponentCookie == nil {
		t.Fatal("opponent should receive a play_session cookie")
	}
	if got := decodeRole(t, opponentCookie.Value); got != openRole {
		t.Errorf("opponent cookie role = %q, want %q", got, openRole)
	}
	if got := decodeRole(t, creatorCookie.Value); got != creatorRole {
		t.Errorf("creator cookie role = %q, want %q", got, creatorRole)
	}
}

// ---- helpers -------------------------------------------------------------

// containsRoleSignal reports whether the rendered HTML contains the given
// role inside the data-signals payload. The payload is JSON, then
// HTML-attribute-escaped by templ — so the literal "&#34;role&#34;:..."
// form is what actually appears in the body.
func containsRoleSignal(body, role string) bool {
	plain := `"role":"` + role + `"`
	htmlEsc := `&#34;role&#34;:&#34;` + role + `&#34;`
	return strings.Contains(body, plain) || strings.Contains(body, htmlEsc)
}

// openSeat returns the (token, role) of the unclaimed seat in g.
func openSeat(g *game.Game) (token, role string) {
	if !g.PlayMeta.IsSeatClaimed("white") {
		return g.PlayMeta.WhiteToken, "white"
	}
	return g.PlayMeta.BlackToken, "black"
}

func otherRole(role string) string {
	if role == "white" {
		return "black"
	}
	return "white"
}

// decodeRole extracts the "role" claim from a play_session JWT cookie value
// without verifying the signature — tests only care about the payload shape.
func decodeRole(t *testing.T, jwtToken string) string {
	t.Helper()
	// JWT format: header.payload.signature; payload is base64url-encoded JSON
	// containing {"game_id":"...","role":"..."}. Rather than re-implementing
	// JWT parsing, look for the role substring in either segment.
	for _, want := range []string{`"role":"white"`, `"role":"black"`, `"role":"spectator"`} {
		if jwtContains(jwtToken, want) {
			return strings.TrimSuffix(strings.TrimPrefix(want, `"role":"`), `"`)
		}
	}
	t.Fatalf("decodeRole: no role claim found in %q", jwtToken)
	return ""
}

// jwtContains base64-decodes the payload segment of a JWT and reports whether
// it contains the given substring.
func jwtContains(jwtToken, want string) bool {
	parts := strings.Split(jwtToken, ".")
	if len(parts) != 3 {
		return false
	}
	// Base64URL-decode the payload. Use the standard library's URL decoder via
	// a small inline helper to avoid an import for the whole test file.
	dec, err := base64URLDecode(parts[1])
	if err != nil {
		return false
	}
	return strings.Contains(string(dec), want)
}

func base64URLDecode(s string) ([]byte, error) {
	// Pad if needed.
	if pad := len(s) % 4; pad != 0 {
		s += strings.Repeat("=", 4-pad)
	}
	return base64.URLEncoding.DecodeString(s)
}
