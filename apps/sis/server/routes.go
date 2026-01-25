package server

import (
	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/sis/shared/dashboard"
)

func InitRoutes(r *gin.Engine) {
	dashboard.InitRoutes(r)
}
