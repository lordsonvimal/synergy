package dashboard

import (
	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/sis/modules/administration/organization"
	"github.com/lordsonvimal/synergy/apps/sis/shared/appctx"
)

func handleDashboard(c *gin.Context) {
	app := appctx.Get(c)
	metrics, _ := GetMetrics(app.DB)
	orgs, _ := organization.GetOrganizationStats(app.DB)

	RenderDashboardPage(metrics, orgs).Render(
		c.Request.Context(),
		c.Writer,
	)
}
