package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lordsonvimal/synergy/apps/chess/db"
)

type gameEventStore struct{ db *sql.DB }

func (s *gameEventStore) InsertBatch(ctx context.Context, events []*db.GameEvent) error {
	if len(events) == 0 {
		return nil
	}

	placeholders := make([]string, len(events))
	args := make([]any, 0, len(events)*5)
	for i, e := range events {
		placeholders[i] = "(?, ?, ?, ?, ?)"
		args = append(args, e.GameID, e.SessionID, e.EventType, e.Payload, e.OccurredAt)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO game_events (game_id, session_id, event_type, payload, occurred_at)
		 VALUES %s`,
		strings.Join(placeholders, ","),
	), args...); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *gameEventStore) ListByGame(ctx context.Context, gameID string) ([]*db.GameEvent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, game_id, session_id, event_type, payload, occurred_at
		 FROM game_events WHERE game_id = ? ORDER BY occurred_at ASC`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*db.GameEvent
	for rows.Next() {
		e := &db.GameEvent{}
		if err := rows.Scan(
			&e.ID, &e.GameID, &e.SessionID, &e.EventType, &e.Payload, &e.OccurredAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
