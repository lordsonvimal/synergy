package sqlite

import (
	"context"
	"database/sql"

	"github.com/lordsonvimal/synergy/apps/chess/db"
)

type chatStore struct{ db *sql.DB }

func (s *chatStore) Insert(ctx context.Context, m *db.ChatMessage) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO chat_messages (session_id, participant_id, body, created_at)
		 VALUES (?, ?, ?, ?)`,
		m.SessionID, m.ParticipantID, m.Body, m.CreatedAt,
	)
	return err
}

func (s *chatStore) ListBySession(ctx context.Context, sessionID string, limit int) ([]*db.ChatMessage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, participant_id, body, created_at
		 FROM chat_messages WHERE session_id = ?
		 ORDER BY created_at ASC LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*db.ChatMessage
	for rows.Next() {
		m := &db.ChatMessage{}
		if err := rows.Scan(&m.ID, &m.SessionID, &m.ParticipantID, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
