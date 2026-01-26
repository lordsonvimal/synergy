package identity

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/sis/shared/appctx"
	"github.com/starfederation/datastar-go/datastar"
)

// Table: users, user_relationships

func handleGetUsers(c *gin.Context) {
	// Implementation for retrieving list of users
	app := appctx.Get(c)
	repo := NewUserRepository(app.DB)
	users, _ := repo.GetAllUsersAcrossOrganizations()
	RenderIndexPage(users).Render(c.Request.Context(), c.Writer)
}

func handleGetUser(c *gin.Context) {
	// Implementation for retrieving a single user
}

func handleCreateUser(c *gin.Context) {
	// Implementation for creating a user
}

func handleShowUserCreateForm(c *gin.Context) {
	// Implementation for showing user creation form
	sse := datastar.NewSSE(c.Writer, c.Request)

	app := appctx.Get(c)
	repo := NewUserRepository(app.DB)
	organizations, err := repo.GetAllOrganizations()
	if err != nil {
		c.Error(err)
		return
	}

	roles, err := repo.GetAllRoles()
	if err != nil {
		c.Error(err)
		return
	}

	buf := new(strings.Builder)
	RenderCreateUserModal(organizations, roles).Render(c.Request.Context(), buf)
	sse.PatchElements(
		buf.String(),
		datastar.WithSelector("#app-modal-root"),
		datastar.WithModeAppend(),
	)
}
