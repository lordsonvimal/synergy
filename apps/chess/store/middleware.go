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
