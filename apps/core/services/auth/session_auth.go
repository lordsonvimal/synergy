package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/sessions"
)

var sessionStore = sessions.NewCookieStore([]byte("super-secret-key"))

type SessionAuthenticator struct{}

func NewSessionAuthenticator() *SessionAuthenticator {
	return &SessionAuthenticator{}
}

func (s *SessionAuthenticator) Login(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "session")
	session.Values["user_id"] = "12345" // Simulated user (replace with actual authentication)
	session.Save(r, w)

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Session created",
	})
}

func (s *SessionAuthenticator) Callback(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"message": "Session authentication successful"}`))
}

func (s *SessionAuthenticator) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "session")
	delete(session.Values, "user_id")
	session.Save(r, w)

	w.Write([]byte(`{"message": "Logged out successfully"}`))
}

func (s *SessionAuthenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "session")

		userID, ok := session.Values["user_id"].(string)
		if !ok || userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
