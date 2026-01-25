package appctx

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

type ctxKeyType string

const ctxKey ctxKeyType = "appctx"

func Middleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		app := &AppContext{
			DB: db,
		}

		c.Set(string(ctxKey), app)
		c.Next()
	}
}

func Get(c *gin.Context) *AppContext {
	val, exists := c.Get(string(ctxKey))
	if !exists {
		panic("AppContext not found in gin.Context")
	}
	return val.(*AppContext)
}
