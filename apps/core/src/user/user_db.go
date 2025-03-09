package user

import (
	"context"
	"fmt"
	"time"

	"github.com/lordsonvimal/synergy/services/db"
)

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuthProvider string

const (
	ProviderGoogle AuthProvider = "google"
	ProviderGitHub AuthProvider = "github"
)

func UserExists(email string, provider AuthProvider) (bool, error) {
	pool := db.GetDB()
	var count int
	query := `
		SELECT COUNT(*) 
		FROM user_auth_providers 
		WHERE user_id = (SELECT id FROM users WHERE email = $1) 
		  AND provider = $2;
	`
	err := pool.QueryRow(context.Background(), query, email, provider).Scan(&count)
	return count > 0, err
}

// CreateUser inserts a user and their authentication provider details with timestamps.
func CreateUser(ctx context.Context, email, provider, displayName, firstName, lastName, hd, picture string, emailVerified bool) error {
	pool := db.GetDB()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	currentTime := time.Now()

	var userID int
	err = tx.QueryRow(ctx, `
        INSERT INTO users (email, created_at, updated_at) 
        VALUES ($1, $2, $2) 
        ON CONFLICT (email) DO UPDATE 
        SET updated_at = EXCLUDED.updated_at 
        RETURNING id
    `, email, currentTime).Scan(&userID)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO user_auth_providers (user_id, provider, display_name, first_name, last_name, hd, picture, email_verified, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
        ON CONFLICT (user_id, provider) DO UPDATE 
        SET display_name = EXCLUDED.display_name,
            first_name = EXCLUDED.first_name,
            last_name = EXCLUDED.last_name,
            hd = EXCLUDED.hd,
            picture = EXCLUDED.picture,
            email_verified = EXCLUDED.email_verified,
            updated_at = EXCLUDED.updated_at
    `, userID, provider, displayName, firstName, lastName, hd, picture, emailVerified, currentTime)

	if err != nil {
		return fmt.Errorf("failed to insert/update auth provider: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
