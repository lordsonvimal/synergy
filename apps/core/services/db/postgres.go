package db

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lordsonvimal/synergy/config"
	"github.com/lordsonvimal/synergy/logger"
)

var (
	dbPool *pgxpool.Pool
	once   sync.Once
)

func InitPostgresDB() {
	once.Do(func() {
		log := logger.GetLogger()
		c, err := config.LoadConfig()

		if err != nil {
			log.Fatal("Failed to load config", map[string]interface{}{"error": err.Error()})
		}

		poolConfig, err := pgxpool.ParseConfig(c.PostgresURL)

		if err != nil {
			log.Fatal("Failed to parse DB config", map[string]interface{}{"error": err.Error()})
		}

		// Set connection pool settings
		poolConfig.MaxConns = c.PostgresMaxConns
		poolConfig.MinConns = c.PostgresMinConns
		poolConfig.MaxConnLifetime = time.Duration(c.PostgresConnMaxLifetime) * time.Second
		poolConfig.MaxConnIdleTime = time.Duration(c.PostgresConnMaxIdleTime) * time.Second

		// Create connection pool
		dbPool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			log.Fatal("Failed to create DB pool", map[string]interface{}{"error": err.Error()})
		}

		log.Info("Connected to PostgreSQL", nil)
	})
}

// GetPostgresPool returns the connection pool instance
func GetPostgresPool() *pgxpool.Pool {
	if dbPool == nil {
		log.Fatal("Database not initialized. Call InitPostgresDB() first.")
	}
	return dbPool
}

// closes the connection pool
func ClosePostgresDB() {
	log := logger.GetLogger()
	if dbPool != nil {
		dbPool.Close()
		log.Info("Database connection closed", nil)
	}
}
