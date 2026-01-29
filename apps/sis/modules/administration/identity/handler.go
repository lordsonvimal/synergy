package identity

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/sis/logger"
	"github.com/lordsonvimal/synergy/apps/sis/shared/appctx"
	"github.com/starfederation/datastar-go/datastar"
)

// Table: users, user_relationships

type UserForm struct {
	Name     string         `form:"name" validate:"required"`
	Email    string         `form:"email" validate:"omitempty,email"`
	Phone    string         `form:"phone"`
	IsActive bool           `form:"is_active"`
	Roles    []UserRoleForm `form:"roles" validate:"dive"`
}

type UserRoleForm struct {
	OrganizationID int `form:"organization_id" validate:"required"`
	RoleID         int `form:"role_id" validate:"required"`
}

func removeDuplicateRoles(roles []UserRoleForm) []UserRoleForm {
	seen := make(map[string]bool)
	var filtered []UserRoleForm

	for _, r := range roles {
		key := fmt.Sprintf("%d-%d", r.OrganizationID, r.RoleID)
		if !seen[key] {
			filtered = append(filtered, r)
			seen[key] = true
		}
	}

	return filtered
}

func PatchUsersTable(sse *datastar.ServerSentEventGenerator, users []UserInfo) error {
	buf := new(strings.Builder)
	RenderUsersList(users).Render(sse.Context(), buf)
	err := sse.PatchElements(
		buf.String(),
	)

	if err != nil {
		return err
	}

	return nil
}

func handleGetUsers(c *gin.Context) {
	// Implementation for retrieving list of users
	app := appctx.Get(c)
	repo := NewUserRepository(app.DB)
	users, err := repo.GetAllUsersAcrossOrganizations()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	RenderIndexPage(users).Render(c.Request.Context(), c.Writer)
}

func handleDeleteUser(c *gin.Context) {
	// Implementation for deleting a user
	userIDParam := c.Param("id")
	var userID int64
	_, err := fmt.Sscanf(userIDParam, "%d", &userID)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid user ID"))
		return
	}

	app := appctx.Get(c)
	repo := NewUserRepository(app.DB)
	if err := repo.DeleteUser(userID); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Create a Server-Sent Event writer
	sse := datastar.NewSSE(c.Writer, c.Request)
	users, _ := repo.GetAllUsersAcrossOrganizations()
	logger.Info(c.Request.Context()).Interface("users", users).Msgf("Fetched %d users after deletion", len(users))
	PatchUsersTable(sse, users)
}

func handleGetUser(c *gin.Context) {
	// Implementation for retrieving a single user
}

func handleCreateUser(c *gin.Context) {
	// Implementation for creating a user
	var form UserForm

	if err := c.ShouldBind(&form); err != nil {
		logger.Error(c.Request.Context()).Err(err).Msg("Unable to read signals")
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	form.Roles = removeDuplicateRoles(form.Roles)

	app := appctx.Get(c)
	repo := NewUserRepository(app.DB)
	if err := repo.CreateUser(&form); err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a Server-Sent Event writer
	sse := datastar.NewSSE(c.Writer, c.Request)
	err := sse.PatchElements(
		"",
		datastar.WithMode(datastar.ElementPatchModeRemove),
		datastar.WithSelector("#create-user-modal"),
	)
	if err != nil {
		c.Error(err)
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	users, _ := repo.GetAllUsersAcrossOrganizations()
	PatchUsersTable(sse, users)
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
