package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/services/auth"
)

type TokenRequest struct {
	Code         string `json:"code" binding:"required"`
	CodeVerifier string `json:"code_verifier" binding:"required"`
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

	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	authenticator.Callback(c.Writer, c.Request)

	c.JSON(http.StatusOK, gin.H{"message": "Redirected from oauth provider."})
}

// Protected route
func LogoutHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "You have access to this protected route!"})
}
