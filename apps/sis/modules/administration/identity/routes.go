package identity

import "github.com/gin-gonic/gin"

func InitRoutes(r *gin.Engine) {
	identityGroup := r.Group("/users")
	{
		identityGroup.GET("", handleGetUsers)
		identityGroup.GET("/new", handleShowUserCreateForm)
		identityGroup.POST("/create", handleCreateUser)
		identityGroup.GET("/:id", handleGetUser)
		identityGroup.DELETE("/:id", handleDeleteUser)
	}
}
