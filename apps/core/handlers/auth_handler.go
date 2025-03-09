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
	}

	authenticator.Login(c.Writer, c.Request)
}

func AuthCallbackHandler(c *gin.Context) {
	if authenticator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authenticator is not defined"})
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

	authenticator.Callback(c.Writer, c.Request)
}

// Protected route
func LogoutHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "You have access to this protected route!"})
}
