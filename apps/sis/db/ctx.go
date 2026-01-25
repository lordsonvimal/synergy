package db

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

type ctxKey string

const dbKey ctxKey = "sqlite_db"

func GetDB(c *gin.Context) *sql.DB {
	return c.MustGet(string(dbKey)).(*sql.DB)
}
