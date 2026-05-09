package sqlite

import (
	"context"
	"database/sql"

	"github.com/lordsonvimal/synergy/apps/chess/db"
)

type gameStore struct{ db *sql.DB }

func (s *gameStore) Create(ctx context.Context, g *db.Game) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO games
		 (id, session_id, white_participant_id, black_participant_id,
		  time_control_ns, increment_ns, variant, initial_fen,
		  status, winner, created_at, ended_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		g.ID, g.SessionID, g.WhiteParticipantID, g.BlackParticipantID,
		g.TimeControlNs, g.IncrementNs, g.Variant, g.InitialFEN,
		g.Status, g.Winner, g.CreatedAt, g.EndedAt,
	)
	return err
}

func (s *gameStore) Get(ctx context.Context, id string) (*db.Game, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, session_id, white_participant_id, black_participant_id,
		        time_control_ns, increment_ns, variant, initial_fen,
		        status, winner, created_at, ended_at
		 FROM games WHERE id = ?`, id)

	g := &db.Game{}
	err := row.Scan(
		&g.ID, &g.SessionID, &g.WhiteParticipantID, &g.BlackParticipantID,
		&g.TimeControlNs, &g.IncrementNs, &g.Variant, &g.InitialFEN,
		&g.Status, &g.Winner, &g.CreatedAt, &g.EndedAt,
	)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (s *gameStore) ListBySession(ctx context.Context, sessionID string) ([]*db.Game, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, white_participant_id, black_participant_id,
		        time_control_ns, increment_ns, variant, initial_fen,
		        status, winner, created_at, ended_at
		 FROM games WHERE session_id = ? ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []*db.Game
	for rows.Next() {
		g := &db.Game{}
		if err := rows.Scan(
			&g.ID, &g.SessionID, &g.WhiteParticipantID, &g.BlackParticipantID,
			&g.TimeControlNs, &g.IncrementNs, &g.Variant, &g.InitialFEN,
			&g.Status, &g.Winner, &g.CreatedAt, &g.EndedAt,
		); err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	return games, rows.Err()
}

func (s *gameStore) UpdateStatus(ctx context.Context, id, status string, winner *string, endedAt *int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE games SET status = ?, winner = ?, ended_at = ? WHERE id = ?`,
		status, winner, endedAt, id,
	)
	return err
}
