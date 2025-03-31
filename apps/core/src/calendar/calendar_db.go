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
	CalendarID  string    `json:"calendar_id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateNewCalendar is a wrapper function for easy calendar creation
func CreateNewCalendar(ctx context.Context, userID int, name, desc string, isDefault bool) (string, error) {
	cal := &Calendar{
		UserID:      userID,
		Name:        name,
		Description: desc,
		IsDefault:   isDefault,
	}
	return cal.Create(ctx)
}

// Create inserts a new calendar into ScyllaDB
func (cal *Calendar) Create(ctx context.Context) (string, error) {
	// Add context timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get Scylla session from context
	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	// Generate UUID for calendar ID
	cal.CalendarID = uuid.New().String()
	now := time.Now().UTC()
	cal.CreatedAt = now
	cal.UpdatedAt = now

	// CQL query with parameterized inputs
	query := `INSERT INTO calendars (user_id, calendar_id, is_default, name, description, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	// Execute the query
	err = session.Query(query,
		cal.UserID,
		cal.CalendarID,
		cal.IsDefault,
		cal.Name,
		cal.Description,
		cal.CreatedAt,
		cal.UpdatedAt,
	).Exec()

	if err != nil {
		return "", fmt.Errorf("failed to create calendar: %w", err)
	}

	fmt.Println("Calendar created:", cal.CalendarID)
	return cal.CalendarID, nil
}

// GetCalendarByID fetches a calendar by user_id and calendar_id
func GetCalendarByID(ctx context.Context, userID int, calendarID string) (*Calendar, error) {
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

// UpdateCalendar updates an existing calendar in ScyllaDB
func (cal *Calendar) Update(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	cal.UpdatedAt = time.Now().UTC()

	query := `UPDATE calendars 
			  SET name = ?, description = ?, is_default = ?, updated_at = ?
			  WHERE user_id = ? AND calendar_id = ?`

	err = session.Query(query,
		cal.Name,
		cal.Description,
		cal.IsDefault,
		cal.UpdatedAt,
		cal.UserID,
		cal.CalendarID,
	).Exec()

	if err != nil {
		return fmt.Errorf("failed to update calendar: %w", err)
	}

	fmt.Println("Calendar updated:", cal.CalendarID)
	return nil
}

// DeleteCalendar deletes a calendar by user_id and calendar_id
func DeleteCalendar(ctx context.Context, userID int, calendarID string) error {
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
