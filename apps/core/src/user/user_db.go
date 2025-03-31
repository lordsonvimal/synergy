package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lordsonvimal/synergy/services/db"
	"github.com/lordsonvimal/synergy/src/calendar"
)

type UserAuthProvider struct {
	ID          int     `json:"id"`
	Email       string  `json:"email"`
	DisplayName *string `json:"display_name"`
	Picture     *string `json:"picture"`
}

type UserAuthInfo struct {
	Email         string
	Provider      string
	DisplayName   string
	FirstName     string
	LastName      string
	HD            string
	Picture       string
	EmailVerified bool
}

type AuthProvider string

const (
	ProviderGoogle AuthProvider = "google"
	ProviderGitHub AuthProvider = "github"
)

func GetUserID(ctx context.Context, email string, provider AuthProvider) (bool, int, error) {
	pool, err := db.GetPgxPoolFromCtx(ctx)
	if err != nil {
		return false, 0, err
	}

	var userID int

	query := `
		SELECT u.id
		FROM users u
		JOIN user_auth_providers uap ON u.id = uap.user_id
		WHERE u.email = $1 AND uap.provider = $2;
	`

	err = pool.QueryRow(ctx, query, email, provider).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, 0, nil // User not found
		}
		return false, 0, fmt.Errorf("failed to fetch user ID: %w", err)
	}

	return true, userID, nil // User exists, return ID
}

func CreateUser(ctx context.Context, userInfo UserAuthInfo) (int, error) {
	pool, err := db.GetPgxPoolFromCtx(ctx)
	if err != nil {
		return 0, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to start transaction: %w", err)
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
    `, userInfo.Email, currentTime).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert user: %w", err)
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
    `, userID, userInfo.Provider, userInfo.DisplayName, userInfo.FirstName, userInfo.LastName, userInfo.HD, userInfo.Picture, userInfo.EmailVerified, currentTime)

	if err != nil {
		return 0, fmt.Errorf("failed to insert/update auth provider: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		_ = tx.Rollback(ctx) // Ensure rollback on commit failure
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	calendar.CreateCalendar(ctx, userID, "My Calendar", "", true)

	return userID, nil
}

func GetUserByID(ctx context.Context, userID int) (*UserAuthProvider, error) {
	pool, err := db.GetPgxPoolFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	query := `
	SELECT 
		u.id, u.email, uap.display_name, uap.picture
	FROM users u
	LEFT JOIN user_auth_providers uap ON u.id = uap.user_id
	WHERE u.id = $1;
	`

	row := pool.QueryRow(context.Background(), query, userID)

	// Scan into struct
	var user UserAuthProvider
	err = row.Scan(
		&user.ID, &user.Email, &user.DisplayName,
		&user.Picture,
	)
	if err != nil {
		return &user, err
	}

	return &user, nil
}
