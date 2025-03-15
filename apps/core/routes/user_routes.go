package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/handlers"
	"github.com/lordsonvimal/synergy/services/auth"
)

func RegisterUserRoutes(router *gin.RouterGroup) {
	userGroup := router.Group("/users")
	userGroup.Use(auth.JWTAuthMiddleware())
	{
		userGroup.GET("/:id", handlers.GetUserByID)
	}
}
