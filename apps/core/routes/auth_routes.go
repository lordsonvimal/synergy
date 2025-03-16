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
	auth.SetAuthenticator(oauthAuthenticator)
	authGroup := router.Group("/auth")
	{
		authGroup.GET("/redirect", handlers.AuthRedirectHandler)
		authGroup.GET("/login", auth.JWTAuthMiddleware(), handlers.LoginHandler)
		authGroup.POST("/logout", handlers.LogoutHandler)
		authGroup.GET("/callback", handlers.AuthCallbackHandler)
	}
}
