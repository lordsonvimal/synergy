package routes

import (
	"github.com/gin-gonic/gin"
)

func RegisterAPIRoutes(router *gin.Engine) {
	apiV1 := router.Group("/api/v1")
	{
		RegisterAuthRoutes(apiV1)
		RegisterUserRoutes(apiV1)
	}
}
