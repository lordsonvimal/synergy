package sqlite

import (
	"context"
	"database/sql"

	"github.com/lordsonvimal/synergy/apps/chess/db"
)

type participantStore struct{ db *sql.DB }

func (s *participantStore) Create(ctx context.Context, p *db.Participant) error {
	claimed := 0
	if p.Claimed {
		claimed = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO participants (id, session_id, role, display_name, token, joined_at, left_at, claimed)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.SessionID, p.Role, p.DisplayName, p.Token, p.JoinedAt, p.LeftAt, claimed,
	)
	return err
}

func (s *participantStore) GetByToken(ctx context.Context, token string) (*db.Participant, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, session_id, role, display_name, token, joined_at, left_at, claimed
		 FROM participants WHERE token = ?`, token)

	p := &db.Participant{}
	var claimed int64
	err := row.Scan(&p.ID, &p.SessionID, &p.Role, &p.DisplayName, &p.Token, &p.JoinedAt, &p.LeftAt, &claimed)
	if err != nil {
		return nil, err
	}
	p.Claimed = claimed != 0
	return p, nil
}

func (s *participantStore) ListBySession(ctx context.Context, sessionID string) ([]*db.Participant, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, role, display_name, token, joined_at, left_at, claimed
		 FROM participants WHERE session_id = ? ORDER BY joined_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*db.Participant
	for rows.Next() {
		p := &db.Participant{}
		var claimed int64
		if err := rows.Scan(
			&p.ID, &p.SessionID, &p.Role, &p.DisplayName,
			&p.Token, &p.JoinedAt, &p.LeftAt, &claimed,
		); err != nil {
			return nil, err
		}
		p.Claimed = claimed != 0
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func (s *participantStore) SetClaimed(ctx context.Context, id string, claimedAt int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE participants SET claimed = 1, joined_at = ? WHERE id = ?`,
		claimedAt, id,
	)
	return err
}
