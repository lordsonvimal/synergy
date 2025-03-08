package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/lordsonvimal/synergy/config"
)

// ValidateSchema checks if DB schema is in sync with migrations
func ValidateSchema() error {
	c, err := config.LoadConfig()

	// Open a separate *sql.DB connection for migrations
	db, err := sql.Open("pgx", c.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect for migrations: %w", err)
	}
	defer db.Close()

	// Setup migration driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations", "postgres", driver,
	)
	if err != nil {
		return fmt.Errorf("migration initialization failed: %w", err)
	}

	// Check if database is at latest version
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to check migration version: %w", err)
	}

	if dirty {
		return fmt.Errorf("database is in a dirty state (migration failed previously)")
	}

	log.Printf("Database is at version: %d", version)
	return nil
}

// RunMigrations applies database migrations
func RunMigrations() error {
	c, err := config.LoadConfig()

	// Open *sql.DB connection
	db, err := sql.Open("pgx", c.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect for migrations: %w", err)
	}
	defer db.Close()

	// Migration driver setup
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("migration initialization failed: %w", err)
	}

	// Apply migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("Database migrations applied successfully")
	return nil
}
