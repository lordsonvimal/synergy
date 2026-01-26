package server

import (
	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/sis/modules/administration/identity"
	"github.com/lordsonvimal/synergy/apps/sis/modules/administration/organization"
	"github.com/lordsonvimal/synergy/apps/sis/shared/dashboard"
)

func InitRoutes(r *gin.Engine) {
	dashboard.InitRoutes(r)
	organization.InitRoutes(r)
	identity.InitRoutes(r)
}
