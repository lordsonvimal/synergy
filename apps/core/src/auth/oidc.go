package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

type OIDCAuthenticator struct {
	provider *oidc.Provider
	config   *oauth2.Config
}

func NewOIDCAuthenticator() (*OIDCAuthenticator, error) {
	provider, err := oidc.NewProvider(context.Background(), os.Getenv("OIDC_ISSUER"))
	if err != nil {
		return nil, err
	}

	return &OIDCAuthenticator{
		provider: provider,
		config: &oauth2.Config{
			ClientID:     os.Getenv("OIDC_CLIENT_ID"),
			ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
			Endpoint:     provider.Endpoint(),
			RedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
	}, nil
}

func (o *OIDCAuthenticator) Login(w http.ResponseWriter, r *http.Request) {
	url := o.config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusFound)
}

func (o *OIDCAuthenticator) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := o.config.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	idToken, err := o.provider.Verifier(&oidc.Config{ClientID: o.config.ClientID}).Verify(context.Background(), token.Extra("id_token").(string))
	if err != nil {
		http.Error(w, "Invalid ID token", http.StatusUnauthorized)
		return
	}

	claims := map[string]interface{}{}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(claims)
}

func (o *OIDCAuthenticator) Logout(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"message": "OIDC logout handled at the identity provider"}`))
}

func (o *OIDCAuthenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "OIDC authentication not implemented for middleware", http.StatusNotImplemented)
	})
}
