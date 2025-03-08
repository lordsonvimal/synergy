package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Home route
func LoginHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Welcome! Visit /auth/login to authenticate."})
}

func AuthCallbackHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Redirected from oauth provider."})
}

// Protected route
func LogoutHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "You have access to this protected route!"})
}
