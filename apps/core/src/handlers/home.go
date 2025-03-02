package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Home route
func HomeHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Welcome! Visit /auth/login to authenticate."})
}

// Protected route
func ProtectedHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "You have access to this protected route!"})
}
