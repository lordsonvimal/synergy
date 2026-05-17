package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lordsonvimal/synergy/apps/chess/db"
)

type moveStore struct{ db *sql.DB }

func (s *moveStore) InsertBatch(ctx context.Context, moves []*db.Move) error {
	if len(moves) == 0 {
		return nil
	}

	placeholders := make([]string, len(moves))
	args := make([]any, 0, len(moves)*13)
	for i, m := range moves {
		placeholders[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(args,
			m.GameID, m.SessionID, m.Seq, m.UCI, m.SAN, m.FEN,
			m.MoveNumber, m.Color, m.WRemNs, m.BRemNs,
			m.LagCompNs, m.ThinkTimeNs, m.PlayedAt,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`INSERT OR IGNORE INTO moves
		 (game_id, session_id, seq, uci, san, fen, move_number, color,
		  w_rem_ns, b_rem_ns, lag_comp_ns, think_time_ns, played_at)
		 VALUES %s`,
		strings.Join(placeholders, ","),
	), args...); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *moveStore) DeleteFromSeq(ctx context.Context, gameID string, fromSeq uint64) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM moves WHERE game_id = ? AND seq >= ?`,
		gameID, fromSeq,
	)
	return err
}

func (s *moveStore) ListByGame(ctx context.Context, gameID string) ([]*db.Move, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, game_id, session_id, seq, uci, san, fen, move_number,
		        color, w_rem_ns, b_rem_ns, lag_comp_ns, think_time_ns, played_at
		 FROM moves WHERE game_id = ? ORDER BY seq ASC`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*db.Move
	for rows.Next() {
		m := &db.Move{}
		if err := rows.Scan(
			&m.ID, &m.GameID, &m.SessionID, &m.Seq, &m.UCI, &m.SAN, &m.FEN,
			&m.MoveNumber, &m.Color, &m.WRemNs, &m.BRemNs,
			&m.LagCompNs, &m.ThinkTimeNs, &m.PlayedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
