package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/services/auth"
)

type TokenRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

var authenticator *auth.OAuthAuthenticator

// SetAuthenticator initializes the authenticator (called in main)
func SetAuthenticator(a *auth.OAuthAuthenticator) {
	authenticator = a
}

// Home route
func LoginHandler(c *gin.Context) {
	if authenticator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authenticator is not defined"})
		return
	}

	authenticator.Login(c.Writer, c.Request)
}

func AuthCallbackHandler(c *gin.Context) {
	if authenticator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authenticator is not defined"})
		return
	}

	// Extract query parameters
	req := TokenRequest{
		Code:  c.Query("code"),
		State: c.Query("state"),
	}

	// Validate required fields
	if req.Code == "" || req.State == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request, missing code or state"})
		return
	}

	// Retrieve cookies
	stateCookie, err := c.Cookie("oauth_state")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid state parameter"})
		return
	}

	codeVerifier, err := c.Cookie("code_verifier")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Code verifier not found"})
		return
	}

	// Authenticate user
	ctx := c.Request.Context()
	authResult, err := authenticator.Callback(ctx, auth.AuthRequest{
		Code:         req.Code,
		State:        req.State,
		StoredState:  stateCookie,
		CodeVerifier: codeVerifier,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Set JWT in HttpOnly Cookie for web users
	c.SetCookie("access_token", authResult.JWTToken, 3600, "/", "", false, true)

	// Return token for Mobile/SPAs
	c.JSON(http.StatusOK, gin.H{"user_id": authResult.UserID})
}

// Protected route
func LogoutHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "You have access to this protected route!"})
}
