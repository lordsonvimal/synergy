package sqlite

import (
	"database/sql"

	"github.com/lordsonvimal/synergy/apps/chess/db"
)

type Repository struct {
	sqlDB *sql.DB
}

func New(sqlDB *sql.DB) db.Repository {
	return &Repository{sqlDB: sqlDB}
}

func (r *Repository) Sessions() db.SessionStore     { return &sessionStore{r.sqlDB} }
func (r *Repository) Games() db.GameStore           { return &gameStore{r.sqlDB} }
func (r *Repository) Moves() db.MoveStore           { return &moveStore{r.sqlDB} }
func (r *Repository) GameEvents() db.GameEventStore { return &gameEventStore{r.sqlDB} }
func (r *Repository) Participants() db.ParticipantStore {
	return &participantStore{r.sqlDB}
}
func (r *Repository) Chat() db.ChatStore { return &chatStore{r.sqlDB} }
func (r *Repository) Close() error       { return r.sqlDB.Close() }
