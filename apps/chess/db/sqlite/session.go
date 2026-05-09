package sqlite

import (
	"context"
	"database/sql"

	"github.com/lordsonvimal/synergy/apps/chess/db"
)

type sessionStore struct{ db *sql.DB }

func (s *sessionStore) Create(ctx context.Context, sess *db.Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, session_type, title, status, created_at, started_at, ended_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.SessionType, sess.Title, sess.Status,
		sess.CreatedAt, sess.StartedAt, sess.EndedAt,
	)
	return err
}

func (s *sessionStore) Get(ctx context.Context, id string) (*db.Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, session_type, title, status, created_at, started_at, ended_at
		 FROM sessions WHERE id = ?`, id)

	sess := &db.Session{}
	err := row.Scan(&sess.ID, &sess.SessionType, &sess.Title, &sess.Status,
		&sess.CreatedAt, &sess.StartedAt, &sess.EndedAt)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *sessionStore) UpdateStatus(ctx context.Context, id, status string, endedAt *int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET status = ?, ended_at = ? WHERE id = ?`,
		status, endedAt, id,
	)
	return err
}
