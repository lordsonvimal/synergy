package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lordsonvimal/synergy/services/db"
)

// Event struct for CRUD operations
type Event struct {
	EventID     uuid.UUID `json:"event_id"`
	CalendarID  string    `json:"calendar_id"`
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

func CreateEvent(ctx context.Context, event *Event) (string, error) {
	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get scylla session: %w", err)
	}

	eventID := uuid.New()
	now := time.Now().UTC()

	query := `
	INSERT INTO events (
		calendar_id, event_id, title, description, location, start_time, end_time, 
		is_all_day, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	err = session.Query(
		query,
		event.CalendarID,
		eventID,
		event.Title,
		event.Description,
		event.Location,
		event.StartTime,
		event.EndTime,
		event.IsAllDay,
		now,
		now,
	).Exec()

	if err != nil {
		return "", fmt.Errorf("failed to insert event: %w", err)
	}

	return eventID.String(), nil
}

func GetEventByID(ctx context.Context, calendarID int, eventID string) (*EventResponse, error) {
	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get scylla session: %w", err)
	}

	var e Event

	query := `
	SELECT event_id, title, description, location, start_time, end_time, 
	       is_all_day, created_at, updated_at
	FROM events
	WHERE calendar_id = ? AND event_id = ?
	LIMIT 1
	`

	err = session.Query(query, calendarID, eventID).Scan(
		&e.EventID, &e.Title, &e.Description, &e.Location,
		&e.StartTime, &e.EndTime, &e.IsAllDay,
		&e.CreatedAt, &e.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}

	response := &EventResponse{
		EventID:     e.EventID.String(),
		Title:       e.Title,
		Description: e.Description,
		Location:    e.Location,
		StartTime:   e.StartTime.Format(time.RFC3339),
		EndTime:     e.EndTime.Format(time.RFC3339),
		IsAllDay:    e.IsAllDay,
		CreatedAt:   e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   e.UpdatedAt.Format(time.RFC3339),
	}

	return response, nil
}

func UpdateEvent(ctx context.Context, calendarID int, eventID string, updatedEvent *Event) error {
	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get scylla session: %w", err)
	}

	now := time.Now().UTC()

	query := `
	UPDATE events
	SET title = ?, description = ?, location = ?, 
	    start_time = ?, end_time = ?, is_all_day = ?, updated_at = ?
	WHERE calendar_id = ? AND event_id = ?
	`

	err = session.Query(
		query,
		updatedEvent.Title, updatedEvent.Description, updatedEvent.Location,
		updatedEvent.StartTime, updatedEvent.EndTime, updatedEvent.IsAllDay,
		now, calendarID, eventID,
	).Exec()

	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

func DeleteEvent(ctx context.Context, calendarID int, eventID string) error {
	session, err := db.GetScyllaSessionFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("failed to get scylla session: %w", err)
	}

	query := `DELETE FROM events WHERE calendar_id = ? AND event_id = ?`
	err = session.Query(query, calendarID, eventID).Exec()

	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}
