package db

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/lordsonvimal/synergy/config"
	"github.com/lordsonvimal/synergy/logger"
)

var (
	scyllaSession *gocql.Session
	scyllaOnce    sync.Once
)

// InitScyllaDB initializes the ScyllaDB session with a connection pool
func InitScyllaDB() {
	scyllaOnce.Do(func() {
		log := logger.GetLogger()
		c, err := config.LoadConfig()

		if err != nil {
			log.Fatal("Failed to load config", map[string]any{"error": err.Error()})
		}

		// Create Scylla cluster configuration
		cluster := gocql.NewCluster(c.ScyllaHosts...)
		cluster.Keyspace = c.ScyllaKeyspace
		cluster.Consistency = gocql.Quorum
		cluster.Timeout = time.Duration(c.ScyllaTimeout) * time.Second
		cluster.ConnectTimeout = time.Duration(c.ScyllaConnectTimeout) * time.Second

		// Connection pooling settings
		cluster.NumConns = int(c.ScyllaMaxConns) // Max connections per host
		cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
		cluster.ReconnectInterval = 5 * time.Second // Auto-reconnect interval

		// Establish connection
		session, err := cluster.CreateSession()
		if err != nil {
			log.Fatal("Failed to connect to ScyllaDB", map[string]any{"error": err.Error()})
		}

		scyllaSession = session
		log.Info("Connected to ScyllaDB", nil)
	})
}

// GetScyllaSession returns the ScyllaDB session
func GetScyllaSession() *gocql.Session {
	if scyllaSession == nil {
		log.Fatal("ScyllaDB not initialized. Call InitScyllaDB() first.")
	}
	return scyllaSession
}

func GetScyllaSessionFromCtx(ctx context.Context) (*gocql.Session, error) {
	db, ok := ctx.Value(DBKey).(*DBs)
	if !ok {
		return nil, errors.New("session not found in context. set it in middleware")
	}
	return db.ScyllaSession, nil
}

// CloseScyllaDB closes the ScyllaDB session
func CloseScyllaDB() {
	log := logger.GetLogger()
	if scyllaSession != nil {
		scyllaSession.Close()
		log.Info("ScyllaDB connection closed", nil)
	}
}
