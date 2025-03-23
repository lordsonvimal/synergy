package db

import (
	"sync"

	"github.com/gocql/gocql"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBs struct {
	PgPool        *pgxpool.Pool
	ScyllaSession *gocql.Session
}

var (
	dbs     *DBs
	dbsOnce sync.Once
)

type dbKey string

const DBKey dbKey = "db"

// InitDBs initializes the DBs
func InitDBs(pgPool *pgxpool.Pool, scyllaSession *gocql.Session) {
	dbsOnce.Do(func() {
		dbs = &DBs{
			PgPool:        pgPool,
			ScyllaSession: scyllaSession,
		}
	})
}

// GetDBs returns the singleton container
func GetDBs() *DBs {
	if dbs == nil {
		panic("DBs not initialized. Call InitDBs() first.")
	}
	return dbs
}
