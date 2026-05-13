package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/server"
	"github.com/lordsonvimal/synergy/apps/chess/store"
)

const testCSRFToken = "test-csrf-token-1234567890abcdef"

// newTestRouter builds a gin router that mirrors main.go middleware setup,
// suitable for handler tests.
func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	t.Setenv("DATA_DIR", t.TempDir())
	gin.SetMode(gin.TestMode)

	router := gin.New()
	gameStore := store.NewGameStore()

	router.Use(store.StoreContext(gameStore))
	router.Use(server.CSRFMiddleware())

	server.InitRoutes(router)
	return router
}

// postWithCSRF builds a POST request that satisfies the CSRF middleware:
// a matching token is placed both in the cookie and in the form body.
func postWithCSRF(target string, form url.Values) *http.Request {
	form.Set("_csrf", testCSRFToken)
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: testCSRFToken})
	return req
}

// ---- ShowGameModes --------------------------------------------------------

func TestShowGameModesReturns200(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET / status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestShowGameModesContainsGameModeButton(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "game-mode-button-0") {
		t.Error("response body should contain a game-mode-button data-testid")
	}
}

func TestShowGameModesErrorQuery(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/?error=game_not_found", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /?error=… status: got %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "error-banner") {
		t.Error("error query param should produce an error-banner in the response")
	}
}

// ---- CreateGame -----------------------------------------------------------

func TestCreateGameRedirectsToGame(t *testing.T) {
	r := newTestRouter(t)
	req := postWithCSRF("/solo", url.Values{"mode": {"Standard 5+2"}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("POST /game status: got %d, want %d (redirect)", w.Code, http.StatusSeeOther)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "/solo/") {
		t.Errorf("Location header should start with /solo/, got %q", loc)
	}
}

func TestCreateGameInvalidModeRedirectsWithError(t *testing.T) {
	r := newTestRouter(t)
	req := postWithCSRF("/solo", url.Values{"mode": {"not-a-real-mode"}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("POST /game with bad mode: got %d, want redirect", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "invalid_mode") {
		t.Errorf("should redirect with invalid_mode error, got %q", w.Header().Get("Location"))
	}
}

// ---- ShowGame -------------------------------------------------------------

// createGameViaHTTP creates a game through the handler and returns its ID.
func createGameViaHTTP(t *testing.T, r *gin.Engine) string {
	t.Helper()
	req := postWithCSRF("/solo", url.Values{"mode": {"Standard 5+2"}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("game creation failed: status %d", w.Code)
	}
	loc := w.Header().Get("Location")
	// loc is /solo/<uuid>
	parts := strings.Split(loc, "/")
	if len(parts) < 3 {
		t.Fatalf("unexpected Location: %q", loc)
	}
	return parts[len(parts)-1]
}

func TestShowGameReturns200(t *testing.T) {
	r := newTestRouter(t)
	gameID := createGameViaHTTP(t, r)

	req := httptest.NewRequest(http.MethodGet, "/solo/"+gameID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /solo/:id status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestShowGameContainsChessboard(t *testing.T) {
	r := newTestRouter(t)
	gameID := createGameViaHTTP(t, r)

	req := httptest.NewRequest(http.MethodGet, "/solo/"+gameID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), `data-testid="chessboard"`) {
		t.Error("game page should contain a chessboard element with data-testid")
	}
}

func TestShowGameNotFoundRedirects(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/solo/does-not-exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("GET /solo/nonexistent status: got %d, want redirect", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "game_not_found") {
		t.Errorf("redirect should include game_not_found, got %q", w.Header().Get("Location"))
	}
}

// ---- SelectSquare ---------------------------------------------------------

func TestSelectSquareReturns200(t *testing.T) {
	r := newTestRouter(t)
	gameID := createGameViaHTTP(t, r)

	// Select square 12 (e2) — White's e-pawn starting square.
	// Must include the _csrf token in the form body; the CSRF middleware rejects
	// POST requests that have neither a matching form field nor a same-origin header.
	req := postWithCSRF("/solo/"+gameID+"/select/12", url.Values{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// The handler either writes SSE signals (200) or returns immediately.
	// We only check that it does not 5xx/4xx.
	if w.Code >= http.StatusBadRequest {
		t.Errorf("POST /solo/:id/select/:sq status: got %d, want < 400", w.Code)
	}
}

func TestSelectSquareInvalidGameReturns404(t *testing.T) {
	r := newTestRouter(t)
	req := postWithCSRF("/solo/bad-id/select/12", url.Values{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("SELECT on missing game: got %d, want %d", w.Code, http.StatusNotFound)
	}
}
