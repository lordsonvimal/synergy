package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lordsonvimal/synergy/services/db"
)

// Calendar struct for creating and retrieving calendars
type Calendar struct {
	CalendarID  uuid.UUID `json:"calendar_id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create inserts a new calendar into ScyllaDB
func CreateCalendar(ctx context.Context, userID int, name, desc string, isDefault bool) (uuid.UUID, error) {
	// Add context timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get Scylla session from context
	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Generate UUID for calendar ID
	calendarID := uuid.New()
	now := time.Now().UTC()

	// CQL query with parameterized inputs
	query := `INSERT INTO calendars (user_id, calendar_id, is_default, name, description, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	// Execute the query
	err = session.Query(query,
		userID,
		calendarID,
		isDefault,
		name,
		desc,
		now,
		now,
	).Exec()

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create calendar: %w", err)
	}

	fmt.Println("Calendar created:", calendarID)
	return calendarID, nil
}

// GetCalendarByID fetches a calendar by user_id and calendar_id
func GetCalendarByID(ctx context.Context, userID int, calendarID uuid.UUID) (*Calendar, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	query := `SELECT calendar_id, user_id, is_default, name, description, created_at, updated_at 
			  FROM calendars WHERE user_id = ? AND calendar_id = ? LIMIT 1`

	var cal Calendar

	err = session.Query(query, userID, calendarID).Scan(
		&cal.CalendarID,
		&cal.UserID,
		&cal.IsDefault,
		&cal.Name,
		&cal.Description,
		&cal.CreatedAt,
		&cal.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendar: %w", err)
	}

	return &cal, nil
}

func UpdateCalendar(ctx context.Context, userID int, calendarID uuid.UUID, name string, desc string, isDefault bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	now := time.Now().UTC()

	query := `UPDATE calendars 
			  SET name = ?, description = ?, is_default = ?, updated_at = ?
			  WHERE user_id = ? AND calendar_id = ?`

	err = session.Query(query,
		name,
		desc,
		isDefault,
		now,
		userID,
		calendarID,
	).Exec()

	if err != nil {
		return fmt.Errorf("failed to update calendar: %w", err)
	}

	fmt.Println("Calendar updated:", calendarID)
	return nil
}

func DeleteCalendar(ctx context.Context, userID int, calendarID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	query := `DELETE FROM calendars WHERE user_id = ? AND calendar_id = ?`

	err = session.Query(query, userID, calendarID).Exec()

	if err != nil {
		return fmt.Errorf("failed to delete calendar: %w", err)
	}

	fmt.Println("Calendar deleted:", calendarID)
	return nil
}
