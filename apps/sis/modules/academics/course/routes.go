package course

import "github.com/gin-gonic/gin"

func InitRoutes(r *gin.Engine) {
	// Course routes would be initialized here
	courseGroup := r.Group("/courses")
	{
		courseGroup.GET("/", handleGetCourses)
		courseGroup.GET("/:id", handleGetCourse)
		courseGroup.POST("/create", handleCreateCourse)
	}
}
