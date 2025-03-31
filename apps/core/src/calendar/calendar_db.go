package calendar

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/lordsonvimal/synergy/services/db"
)

const timeout = 5 * time.Second

const (
	queryInsertCalendar = `INSERT INTO calendars (user_id, calendar_id, is_default, name, description, created_at, updated_at) 
						   VALUES (?, ?, ?, ?, ?, ?, ?)`

	queryGetCalendar = `SELECT calendar_id, user_id, is_default, name, description, created_at, updated_at 
						FROM calendars WHERE user_id = ? AND calendar_id = ? LIMIT 1`

	queryUpdateCalendar = `UPDATE calendars 
						   SET name = ?, description = ?, is_default = ?, updated_at = ?
						   WHERE user_id = ? AND calendar_id = ? IF EXISTS`

	queryDeleteCalendar = `DELETE FROM calendars WHERE user_id = ? AND calendar_id = ? IF EXISTS`
)

type Calendar struct {
	CalendarID  uuid.UUID `json:"calendar_id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func CreateCalendar(ctx context.Context, userID int, name, desc string, isDefault bool) (uuid.UUID, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get session: %w", err)
	}

	calendarID := uuid.New()
	now := time.Now().UTC()

	slog.Info("Creating calendar",
		"user_id", userID,
		"calendar_id", calendarID,
		"name", name,
		"is_default", isDefault)

	err = session.Query(queryInsertCalendar, userID, calendarID, isDefault, name, desc, now, now).
		WithContext(ctx).Exec()

	if err != nil {
		slog.Error("Failed to create calendar", "error", err)
		return uuid.Nil, fmt.Errorf("failed to create calendar: %w", err)
	}

	slog.Info("Calendar created successfully",
		"calendar_id", calendarID,
		"user_id", userID,
		"elapsed_time", time.Since(start))

	return calendarID, nil
}

func ListCalendars(ctx context.Context, userID int, pageSize int, pageState []byte) ([]Calendar, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get session: %w", err)
	}

	query := `SELECT calendar_id, name, description, is_default, created_at, updated_at
			  FROM calendars WHERE user_id = ?`

	iter := session.Query(query, userID).
		WithContext(ctx).
		PageSize(pageSize).
		PageState(pageState).
		Iter()

	var calendars []Calendar
	for {
		var cal Calendar
		if !iter.Scan(&cal.CalendarID, &cal.Name, &cal.Description, &cal.IsDefault, &cal.CreatedAt, &cal.UpdatedAt) {
			break
		}
		calendars = append(calendars, cal)
	}

	if err := iter.Close(); err != nil {
		slog.Error("Failed to close iterator", "error", err)
		return nil, nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	slog.Info("Calendars listed successfully",
		"user_id", userID,
		"page_size", pageSize,
		"num_calendars", len(calendars))

	return calendars, iter.PageState(), nil
}

func GetCalendarByID(ctx context.Context, userID int, calendarID uuid.UUID) (*Calendar, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var cal Calendar

	err = session.Query(queryGetCalendar, userID, calendarID).
		WithContext(ctx).Scan(
		&cal.CalendarID,
		&cal.UserID,
		&cal.IsDefault,
		&cal.Name,
		&cal.Description,
		&cal.CreatedAt,
		&cal.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, errors.New("calendar not found")
		}
		return nil, fmt.Errorf("failed to fetch calendar: %w", err)
	}

	return &cal, nil
}

func UpdateCalendar(ctx context.Context, userID int, calendarID uuid.UUID, name, desc string, isDefault bool) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	now := time.Now().UTC()

	m := map[string]any{}
	err = session.Query(queryUpdateCalendar, name, desc, isDefault, now, userID, calendarID).
		WithContext(ctx).MapScan(m)

	if err != nil {
		return fmt.Errorf("failed to update calendar: %w", err)
	}

	// Check if update was applied
	if len(m) == 0 {
		return fmt.Errorf("calendar not found")
	}

	slog.Info("Calendar updated successfully",
		"calendar_id", calendarID,
		"user_id", userID)

	return nil
}

func DeleteCalendar(ctx context.Context, userID int, calendarID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	m := map[string]any{}
	err = session.Query(queryDeleteCalendar, userID, calendarID).
		WithContext(ctx).MapScan(m)

	if err != nil {
		return fmt.Errorf("failed to delete calendar: %w", err)
	}

	// Check if delete was applied
	if len(m) == 0 {
		return fmt.Errorf("calendar not found")
	}

	slog.Info("Calendar deleted successfully",
		"calendar_id", calendarID,
		"user_id", userID)

	return nil
}
