package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OIDCAuthenticator struct {
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

func NewOIDCAuthenticator() *OIDCAuthenticator {
	providerURL := "https://accounts.google.com" // Change for Okta/Auth0/Azure
	clientID := os.Getenv("OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	redirectURL := "http://localhost:8080/auth/callback"

	provider, err := oidc.NewProvider(context.Background(), providerURL)
	if err != nil {
		log.Fatalf("Failed to get OIDC provider: %v", err)
	}

	return &OIDCAuthenticator{
		oauth2Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}
}

func (o *OIDCAuthenticator) Login(w http.ResponseWriter, r *http.Request) {
	authURL := o.oauth2Config.AuthCodeURL("random-state", oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (o *OIDCAuthenticator) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	code := r.URL.Query().Get("code")

	token, err := o.oauth2Config.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusUnauthorized)
		return
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No ID token found", http.StatusUnauthorized)
		return
	}

	verifiedToken, err := o.verifier.Verify(ctx, idToken)
	if err != nil {
		http.Error(w, "Invalid ID token", http.StatusUnauthorized)
		return
	}

	var claims map[string]interface{}
	if err := verifiedToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Login successful",
		"user":    claims,
		"token":   token.AccessToken,
	})
}

func (o *OIDCAuthenticator) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Logged out successfully."}`))
}
