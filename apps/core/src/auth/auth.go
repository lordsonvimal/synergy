package auth

import (
	"net/http"
)

// Authenticator interface supports both JWT and session authentication.
type Authenticator interface {
	Login(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	Authenticate(next http.Handler) http.Handler
}
