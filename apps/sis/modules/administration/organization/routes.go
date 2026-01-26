package organization

import "github.com/gin-gonic/gin"

func InitRoutes(r *gin.Engine) {
	orgGroup := r.Group("/organizations")
	{
		orgGroup.GET("/", handleGetOrganizations)
		orgGroup.GET("/:id", handleGetOrganization)
		orgGroup.POST("/create", handleCreateOrganization)
	}
}
