package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lordsonvimal/synergy/config"
	"github.com/lordsonvimal/synergy/services/cookie"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// JWTClaims holds user claims for JWT
type JWTClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// OAuthAuthenticator handles OAuth2 authentication
type OAuthAuthenticator struct {
	provider *oidc.Provider
	config   *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// NewOAuthAuthenticator initializes Google OIDC authentication
func NewOAuthAuthenticator() (*OAuthAuthenticator, error) {
	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		return nil, err
	}

	c := config.GetConfig()

	config := &oauth2.Config{
		ClientID:     c.GoogleOauthClientId,
		ClientSecret: c.GoogleOauthClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  c.GoogleOauthRedirectUrl,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: config.ClientID})

	return &OAuthAuthenticator{provider, config, verifier}, nil
}

// Login redirects users to Google's OAuth2 authorization URL
func (o *OAuthAuthenticator) Login(w http.ResponseWriter, r *http.Request) {
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Generate a secure state value
	state := generateState()

	// Store state in a cookie (SPA retrieves it later)
	cookie.SetCookie(w, "oauth_state", state)
	cookie.SetCookie(w, "code_verifier", codeVerifier)

	authURL := o.config.AuthCodeURL(state, oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	// http.Redirect(w, r, authURL, http.StatusFound)
	json.NewEncoder(w).Encode(map[string]string{"url": authURL})
}

// Callback handles the OAuth2 callback and issues a JWT
func (o *OAuthAuthenticator) Callback(w http.ResponseWriter, r *http.Request) {
	// Retrieve state from request
	receivedState := r.URL.Query().Get("state")

	// Retrieve state stored in the cookie
	stateCookie, err := r.Cookie("oauth_state")

	if err != nil || stateCookie.Value != receivedState {
		http.Error(w, "Invalid state parameter", http.StatusUnauthorized)
		return
	}

	codeVerifier, err := r.Cookie("code_verifier")
	if err != nil {
		http.Error(w, "Code verifier not found", http.StatusUnauthorized)
		return
	}

	ctx := context.Background()
	code := r.URL.Query().Get("code")

	token, err := o.config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier.Value))
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No ID token received", http.StatusUnauthorized)
		return
	}

	idToken, err := o.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		http.Error(w, "Invalid ID token", http.StatusUnauthorized)
		return
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		HD            string `json:"hd"` // Hosted domain (Google Workspace users)
	}

	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
		return
	}

	// Check if email is verified
	if !claims.EmailVerified {
		http.Error(w, "Email not verified", http.StatusForbidden)
		return
	}

	// Generate JWT token
	jwtToken, err := generateJWT(claims.Email, claims.HD)
	if err != nil {
		http.Error(w, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	// Set JWT in HttpOnly Cookie for web users
	cookie.SetCookie(w, "access_token", jwtToken)

	// Return token for Mobile/SPAs
	json.NewEncoder(w).Encode(map[string]string{"token": jwtToken})
}

// Logout clears the authentication cookie
func (o *OAuthAuthenticator) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Authenticate middleware for protecting routes
func (o *OAuthAuthenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// generateJWT creates a JWT token
func generateJWT(email string, domain string) (string, error) {
	claims := JWTClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)), // 1 hour expiry
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// generateCodeVerifier creates a random 32-byte code verifier (URL-safe)
func generateCodeVerifier() string {
	verifier := make([]byte, 32)
	_, err := rand.Read(verifier)
	if err != nil {
		panic("Failed to generate code verifier")
	}
	return base64URLEncode(verifier)
}

// generateCodeChallenge generates the code challenge (SHA256 of the verifier)
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64URLEncode(hash[:])
}

// base64URLEncode converts bytes to URL-safe base64 format
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// generateState creates a secure random state string - CSRF
func generateState() string {
	b := make([]byte, 16) // 16 bytes = 128-bit random value
	_, err := rand.Read(b)
	if err != nil {
		panic("Failed to generate state")
	}
	return base64.URLEncoding.EncodeToString(b)
}
