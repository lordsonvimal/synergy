package calendar

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/lordsonvimal/synergy/logger"
	"github.com/lordsonvimal/synergy/services/db"
)

const timeout = 5 * time.Second

const (
	queryInsertCalendar = `INSERT INTO synergy.calendars (user_id, calendar_id, is_default, name, description, created_at, updated_at) 
						   VALUES (?, ?, ?, ?, ?, ?, ?)`

	queryGetCalendar = `SELECT calendar_id, user_id, is_default, name, description, created_at, updated_at 
						FROM synergy.calendars WHERE user_id = ? AND calendar_id = ? LIMIT 1`

	queryListCalendars = `SELECT calendar_id, name, description, is_default, created_at, updated_at
					  FROM synergy.calendars WHERE user_id = ?`

	queryUpdateCalendar = `UPDATE synergy.calendars 
						   SET name = ?, description = ?, is_default = ?, updated_at = ?
						   WHERE user_id = ? AND calendar_id = ? IF EXISTS`

	queryDeleteCalendar = `DELETE FROM synergy.calendars WHERE user_id = ? AND calendar_id = ? IF EXISTS`
)

type Calendar struct {
	CalendarID  gocql.UUID `json:"calendar_id"`
	UserID      int        `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsDefault   bool       `json:"is_default"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func CreateCalendar(ctx context.Context, userID int, name, desc string, isDefault bool) (string, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	calendarID := gocql.TimeUUID()
	now := time.Now().UTC()

	log, ok := ctx.Value(logger.LoggerKey).(logger.Logger)
	if !ok {
		return "", errors.New("logger not found in context. set it in middleware")
	}

	log.Info(ctx, "Creating calendar", map[string]any{
		"user_id":     userID,
		"calendar_id": calendarID,
		"name":        name,
		"is_default":  isDefault})

	err = session.Query(queryInsertCalendar, userID, calendarID, isDefault, name, desc, now, now).
		WithContext(ctx).Exec()

	if err != nil {
		log.Error(ctx, "Failed to create calendar", map[string]any{"error": err})
		return "", fmt.Errorf("failed to create calendar: %w", err)
	}

	log.Info(ctx, "Calendar created successfully", map[string]any{
		"calendar_id":  calendarID,
		"user_id":      userID,
		"elapsed_time": time.Since(start)})

	return calendarID.String(), nil
}

func ListCalendars(ctx context.Context, userID int, pageSize int, pageState []byte) ([]Calendar, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get session: %w", err)
	}

	iter := session.Query(queryListCalendars, userID).
		WithContext(ctx).
		PageSize(pageSize).
		PageState(pageState).
		Iter()

	if iter == nil {
		return nil, nil, fmt.Errorf("failed to create iterator")
	}

	var calendars []Calendar
	for {
		var cal Calendar
		if !iter.Scan(&cal.CalendarID, &cal.Name, &cal.Description, &cal.IsDefault, &cal.CreatedAt, &cal.UpdatedAt) {
			break
		}
		cal.UserID = userID
		calendars = append(calendars, cal)
	}

	log, ok := ctx.Value(logger.LoggerKey).(logger.Logger)
	if !ok {
		return nil, nil, errors.New("logger not found in context. set it in middleware")
	}

	if err := iter.Close(); err != nil {
		log.Error(ctx, "Failed to close iterator", map[string]any{"error": err})
		return nil, nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	log.Info(ctx, "Calendars listed successfully", map[string]any{
		"user_id":       userID,
		"page_size":     pageSize,
		"num_calendars": len(calendars)})

	return calendars, iter.PageState(), nil
}

func GetCalendarByID(ctx context.Context, userID int, calendarID string) (*Calendar, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	calendarUUID, err := gocql.ParseUUID(calendarID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %w", err)
	}

	var cal Calendar

	err = session.Query(queryGetCalendar, userID, calendarUUID).
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

func UpdateCalendar(ctx context.Context, userID int, calendarID string, name, desc string, isDefault bool) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	calendarUUID, err := gocql.ParseUUID(calendarID)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	now := time.Now().UTC()

	m := map[string]any{}
	err = session.Query(queryUpdateCalendar, name, desc, isDefault, now, userID, calendarUUID).
		WithContext(ctx).MapScan(m)

	if err != nil {
		return fmt.Errorf("failed to update calendar: %w", err)
	}

	// Check if update was applied
	if len(m) == 0 {
		return fmt.Errorf("calendar not found")
	}

	log, ok := ctx.Value(logger.LoggerKey).(logger.Logger)
	if !ok {
		return errors.New("logger not found in context. set it in middleware")
	}
	log.Info(ctx, "Calendar updated successfully", map[string]any{
		"calendar_id": calendarID,
		"user_id":     userID})

	return nil
}

func DeleteCalendar(ctx context.Context, userID int, calendarID string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	calendarUUID, err := gocql.ParseUUID(calendarID)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	m := map[string]any{}
	err = session.Query(queryDeleteCalendar, userID, calendarUUID).
		WithContext(ctx).MapScan(m)

	if err != nil {
		return fmt.Errorf("failed to delete calendar: %w", err)
	}

	// Check if delete was applied
	if len(m) == 0 {
		return fmt.Errorf("calendar not found")
	}

	log, ok := ctx.Value(logger.LoggerKey).(logger.Logger)
	if !ok {
		return errors.New("logger not found in context. set it in middleware")
	}

	log.Info(ctx, "Calendar deleted successfully", map[string]any{
		"calendar_id": calendarID,
		"user_id":     userID})

	return nil
}
