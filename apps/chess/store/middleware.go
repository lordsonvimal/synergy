package store

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/ctxkeys"
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
