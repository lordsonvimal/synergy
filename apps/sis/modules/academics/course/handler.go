package course

import (
	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/sis/shared/appctx"
)

// Table: courses

func handleGetCourses(c *gin.Context) {
	ctx := appctx.Get(c)
	repo := NewCourseRepository(ctx.DB)
	courses, _ := repo.GetAllCoursesAcrossOrganizations()
	RenderIndexPage(courses).Render(c.Request.Context(), c.Writer)
}

func handleGetCourse(c *gin.Context) {
	// Implementation for retrieving a single course
}

func handleCreateCourse(c *gin.Context) {
	// Implementation for creating a course
}
