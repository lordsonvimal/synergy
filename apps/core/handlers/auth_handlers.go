package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/services/auth"
	"github.com/lordsonvimal/synergy/services/cookie"
)

type TokenRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

func AuthRedirectHandler(c *gin.Context) {
	authenticator := auth.GetAuthenticator()
	data := authenticator.Redirect(c.Request.Context())

	// Store state in a cookie (SPA retrieves it later)
	cookie.SetCookie(c.Writer, "oauth_state", data.State)
	cookie.SetCookie(c.Writer, "code_verifier", data.CodeVerifier)

	c.JSON(http.StatusOK, gin.H{"url": data.RedirectUrl})
}

// Home route
func LoginHandler(c *gin.Context) {
	authenticator := auth.GetAuthenticator()

	if authenticator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authenticator is not defined"})
		return
	}

	// authenticator.Login(c.Writer, c.Request)
}

func AuthCallbackHandler(c *gin.Context) {
	authenticator := auth.GetAuthenticator()

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
	authResult, err := authenticator.Callback(ctx, auth.AuthCallbackRequest{
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
	cookie.SetCookie(c.Writer, "access_token", authResult.JWTToken)

	// Return token for Mobile/SPAs
	c.JSON(http.StatusOK, gin.H{"user_id": authResult.UserID})
}

func LogoutHandler(c *gin.Context) {
	// Clear cookie
	cookie.SetCookie(c.Writer, "access_token", "")
	c.JSON(http.StatusOK, gin.H{"message": "You have been logged out successfully!"})
}
