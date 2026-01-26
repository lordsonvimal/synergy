package organization

import (
	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/sis/shared/appctx"
)

// Table: organizations

func handleGetOrganizations(c *gin.Context) {
	// Implementation for retrieving list of organizations
	app := appctx.Get(c)
	orgs, _ := GetOrganizationStats(app.DB)

	RenderIndexPage(orgs).Render(
		c.Request.Context(),
		c.Writer,
	)
}

func handleGetOrganization(c *gin.Context) {
	// Implementation for retrieving a single organization
}

func handleCreateOrganization(c *gin.Context) {
	// Implementation for creating an organization
}
