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

// soloMoveTestRouter builds a router suitable for hitting /solo/:id/move.
func soloMoveTestRouter(t *testing.T) (*gin.Engine, *store.GameStore) {
	t.Helper()
	t.Setenv("DATA_DIR", t.TempDir())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	gs := store.NewGameStore()
	r.Use(store.StoreContext(gs))
	r.Use(server.CSRFMiddleware())
	server.InitRoutes(r)
	return r, gs
}

// createSoloGame creates a solo game and returns its ID.
func createSoloGame(t *testing.T, r *gin.Engine) string {
	t.Helper()
	req := postWithCSRF("/solo", url.Values{"mode": {"Unlimited"}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("POST /solo: got %d", w.Code)
	}
	return strings.TrimPrefix(w.Header().Get("Location"), "/solo/")
}

// soloMove issues a POST /solo/:id/move/:from/:to with optional baseSeq.
func soloMove(t *testing.T, r *gin.Engine, gameID string, from, to int, baseSeq int) *httptest.ResponseRecorder {
	t.Helper()
	target := "/solo/" + gameID + "/move/" + itoa(from) + "/" + itoa(to)
	if baseSeq >= 0 {
		target += "?baseSeq=" + itoa(baseSeq)
	}
	req := postWithCSRF(target, url.Values{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func TestSoloMove_AppliedReturns200(t *testing.T) {
	r, _ := soloMoveTestRouter(t)
	gameID := createSoloGame(t, r)
	// e2 = 12, e4 = 28
	w := soloMove(t, r, gameID, 12, 28, 0)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", w.Code, w.Body.String())
	}
}

func TestSoloMove_StaleBaseSeqReturns409(t *testing.T) {
	r, _ := soloMoveTestRouter(t)
	gameID := createSoloGame(t, r)
	// Apply e2-e4 with baseSeq=0 → seq becomes 1.
	w := soloMove(t, r, gameID, 12, 28, 0)
	if w.Code != http.StatusOK {
		t.Fatalf("first move: expected 200, got %d", w.Code)
	}
	// Reuse baseSeq=0 (stale) for the next move → 409 Conflict.
	w = soloMove(t, r, gameID, 52, 36, 0) // e7-e5 with stale base
	if w.Code != http.StatusConflict {
		t.Fatalf("stale baseSeq: expected 409, got %d body=%q", w.Code, w.Body.String())
	}
}

func TestSoloMove_IllegalReturns422(t *testing.T) {
	r, _ := soloMoveTestRouter(t)
	gameID := createSoloGame(t, r)
	// a1 = 0, a8 = 56 — rook can't leap through pieces from start.
	w := soloMove(t, r, gameID, 0, 56, 0)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("illegal move: expected 422, got %d", w.Code)
	}
}

func TestSoloMove_NoBaseSeqAccepted(t *testing.T) {
	// Legacy clients that don't send baseSeq must still work — the parameter
	// is opt-in and absence disables the check.
	r, _ := soloMoveTestRouter(t)
	gameID := createSoloGame(t, r)
	w := soloMove(t, r, gameID, 12, 28, -1)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 without baseSeq, got %d", w.Code)
	}
}
