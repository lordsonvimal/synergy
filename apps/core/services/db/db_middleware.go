package db

import (
	"context"

	"github.com/gin-gonic/gin"
)

// DBMiddleware injects the dbs into the Gin context
func DBMiddleware() gin.HandlerFunc {
	dbs := GetDBs()
	return func(c *gin.Context) {
		// Add the container to the Gin context
		c.Set(string(DBKey), dbs)

		// Inject into Go request context
		reqCtx := context.WithValue(c.Request.Context(), DBKey, dbs)
		c.Request = c.Request.WithContext(reqCtx)

		c.Next()
	}
}
