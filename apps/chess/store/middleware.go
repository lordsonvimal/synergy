package store

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/ctxkeys"
	"github.com/lordsonvimal/synergy/apps/chess/db"
)

func StoreContext(repo GameRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(
			c.Request.Context(),
			ctxkeys.GameRepoKey,
			repo,
		)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func GetRepoFromContext(ctx context.Context) (GameRepository, bool) {
	repo, ok := ctx.Value(ctxkeys.GameRepoKey).(GameRepository)
	return repo, ok
}

func DBRepoContext(repo db.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(
			c.Request.Context(),
			ctxkeys.DBRepoKey,
			repo,
		)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func GetDBRepoFromContext(ctx context.Context) (db.Repository, bool) {
	repo, ok := ctx.Value(ctxkeys.DBRepoKey).(db.Repository)
	return repo, ok
}
