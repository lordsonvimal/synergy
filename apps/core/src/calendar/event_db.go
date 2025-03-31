package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/lordsonvimal/synergy/services/db"
)

type Event struct {
	EventID     uuid.UUID `json:"event_id"`
	CalendarID  uuid.UUID `json:"calendar_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	IsAllDay    bool      `json:"is_all_day"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EventResponse for API responses (to avoid exposing internal fields)
type EventResponse struct {
	EventID     string `json:"event_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Location    string `json:"location"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	IsAllDay    bool   `json:"is_all_day"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func CreateEvent(ctx context.Context, event *Event) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second) // Add timeout
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get Scylla session: %w", err)
	}

	query := `
	INSERT INTO events (
		calendar_id, event_id, title, description, location, start_time, end_time, 
		is_all_day, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	err = session.Query(
		query,
		event.CalendarID,
		event.EventID,
		event.Title,
		event.Description,
		event.Location,
		event.StartTime,
		event.EndTime,
		event.IsAllDay,
		event.CreatedAt,
		event.UpdatedAt,
	).WithContext(ctx).Exec() // Add context to query

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert event: %w", err)
	}

	return event.EventID, nil
}

func GetEventByID(ctx context.Context, calendarID uuid.UUID, eventID uuid.UUID) (*EventResponse, error) {
	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get scylla session: %w", err)
	}

	var e Event

	query := `
	SELECT event_id, calendar_id, title, description, location, start_time, end_time, 
	       is_all_day, created_at, updated_at
	FROM events
	WHERE calendar_id = ? AND event_id = ?
	LIMIT 1
	`

	err = session.Query(query, calendarID, eventID).Scan(
		&e.EventID, &e.CalendarID, &e.Title, &e.Description, &e.Location,
		&e.StartTime, &e.EndTime, &e.IsAllDay, &e.CreatedAt, &e.UpdatedAt,
	)

	// Handle no results separately
	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, fmt.Errorf("event not found")
		}
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}

	response := &EventResponse{
		EventID:     e.EventID.String(),
		Title:       e.Title,
		Description: e.Description,
		Location:    e.Location,
		StartTime:   formatTimestamp(e.StartTime),
		EndTime:     formatTimestamp(e.EndTime),
		IsAllDay:    e.IsAllDay,
		CreatedAt:   formatTimestamp(e.CreatedAt),
		UpdatedAt:   formatTimestamp(e.UpdatedAt),
	}

	return response, nil
}

func UpdateEvent(ctx context.Context, event *Event) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Scylla session: %w", err)
	}

	event.UpdatedAt = time.Now().UTC()

	query := `
	UPDATE events
	SET title = ?, description = ?, location = ?, 
	    start_time = ?, end_time = ?, is_all_day = ?, updated_at = ?
	WHERE calendar_id = ? AND event_id = ?
	`

	err = session.Query(
		query,
		event.Title,
		event.Description,
		event.Location,
		event.StartTime,
		event.EndTime,
		event.IsAllDay,
		event.UpdatedAt,
		event.CalendarID,
		event.EventID,
	).WithContext(ctx).Exec()

	if err != nil {
		if err == gocql.ErrNotFound { // Handle not found error
			return fmt.Errorf("event not found")
		}
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

func DeleteEvent(ctx context.Context, calendarID uuid.UUID, eventID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Scylla session: %w", err)
	}

	query := `DELETE FROM events WHERE calendar_id = ? AND event_id = ?`
	err = session.Query(query, calendarID, eventID).WithContext(ctx).Exec()

	// Handle case where the event doesn't exist
	if err != nil {
		if err == gocql.ErrNotFound {
			return fmt.Errorf("event not found")
		}
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}
