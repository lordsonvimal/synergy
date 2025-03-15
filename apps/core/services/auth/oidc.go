package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lordsonvimal/synergy/config"
	"github.com/lordsonvimal/synergy/services/cookie"
	"github.com/lordsonvimal/synergy/src/user"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleTokenClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HD            string `json:"hd"` // Hosted domain (Google Workspace users)
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// JWTClaims holds user claims for JWT
type JWTClaims struct {
	ID int `json:"id"`
	jwt.RegisteredClaims
}

// OAuthAuthenticator handles OAuth2 authentication
type OAuthAuthenticator struct {
	provider *oidc.Provider
	config   *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

type AuthRequest struct {
	State        string
	Code         string
	StoredState  string
	CodeVerifier string
}

type AuthResult struct {
	UserID   int
	JWTToken string
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
func (o *OAuthAuthenticator) Callback(ctx context.Context, req AuthRequest) (*AuthResult, error) {
	// Validate state
	if req.State != req.StoredState {
		return nil, fmt.Errorf("invalid state parameter")
	}

	// Exchange code for token
	token, err := o.config.Exchange(ctx, req.Code, oauth2.SetAuthURLParam("code_verifier", req.CodeVerifier))
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no ID token received")
	}

	idToken, err := o.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("invalid ID token: %w", err)
	}

	// Parse claims
	claims := GoogleTokenClaims{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Check email verification
	if !claims.EmailVerified {
		return nil, fmt.Errorf("email not verified")
	}

	// Check if user exists or create new user
	exists, userID, err := user.GetUserID(ctx, claims.Email, "google")
	if err != nil {
		return nil, fmt.Errorf("error checking user: %w", err)
	}

	if !exists {
		userID, err = user.CreateUser(ctx, user.UserAuthInfo{
			Email:         claims.Email,
			Provider:      "google",
			DisplayName:   claims.Name,
			FirstName:     claims.GivenName,
			LastName:      claims.FamilyName,
			HD:            claims.HD,
			Picture:       claims.Picture,
			EmailVerified: claims.EmailVerified,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// Generate JWT
	jwtToken, err := generateJWT(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	return &AuthResult{UserID: userID, JWTToken: jwtToken}, nil
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
func generateJWT(userID int) (string, error) {
	fmt.Printf("user id %d", userID)
	claims := JWTClaims{
		ID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)), // 1 hour expiry
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.GetConfig().JWTSecret))
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
