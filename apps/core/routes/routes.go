package routes

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(router *gin.Engine) {
	RegisterAPIRoutes(router)
}
