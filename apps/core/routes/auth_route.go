package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/handlers"
)

// RegisterAuthRoutes registers authentication-related routes
func RegisterAuthRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", handlers.LoginHandler)
		auth.POST("/logout", handlers.LogoutHandler)
		auth.GET("/callback", handlers.AuthCallbackHandler)
	}
}
