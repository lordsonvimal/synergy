package sqlite

import (
	"context"
	"database/sql"

	"github.com/lordsonvimal/synergy/apps/chess/db"
)

type participantStore struct{ db *sql.DB }

func (s *participantStore) Create(ctx context.Context, p *db.Participant) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO participants (id, session_id, role, display_name, token, joined_at, left_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.SessionID, p.Role, p.DisplayName, p.Token, p.JoinedAt, p.LeftAt,
	)
	return err
}

func (s *participantStore) GetByToken(ctx context.Context, token string) (*db.Participant, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, session_id, role, display_name, token, joined_at, left_at
		 FROM participants WHERE token = ?`, token)

	p := &db.Participant{}
	err := row.Scan(&p.ID, &p.SessionID, &p.Role, &p.DisplayName, &p.Token, &p.JoinedAt, &p.LeftAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *participantStore) ListBySession(ctx context.Context, sessionID string) ([]*db.Participant, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, role, display_name, token, joined_at, left_at
		 FROM participants WHERE session_id = ? ORDER BY joined_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*db.Participant
	for rows.Next() {
		p := &db.Participant{}
		if err := rows.Scan(
			&p.ID, &p.SessionID, &p.Role, &p.DisplayName,
			&p.Token, &p.JoinedAt, &p.LeftAt,
		); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}
