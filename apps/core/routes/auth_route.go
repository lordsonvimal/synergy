package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/handlers"
	"github.com/lordsonvimal/synergy/logger"
	"github.com/lordsonvimal/synergy/services/auth"
)

// RegisterAuthRoutes registers authentication-related routes
func RegisterAuthRoutes(router *gin.RouterGroup) {
	oauthAuthenticator, err := auth.NewOAuthAuthenticator()
	if err != nil {
		logger.GetLogger().Fatal("Cannot set authenticator", map[string]interface{}{"error": err})
	}
	handlers.SetAuthenticator(oauthAuthenticator)
	auth := router.Group("/auth")
	{
		auth.POST("/login", handlers.LoginHandler)
		auth.POST("/logout", handlers.LogoutHandler)
		auth.GET("/callback", handlers.AuthCallbackHandler)
	}
}
