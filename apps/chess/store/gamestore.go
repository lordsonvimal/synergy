package store

import (
	"sync"

	"github.com/lordsonvimal/synergy/apps/chess/game"
)

type GameRepository interface {
	Add(*game.Game)
	Get(id string) (*game.Game, bool)
	Delete(id string)
}

type ChessGlobalContext struct {
	Games GameRepository
}

type GameStore struct {
	mu    sync.RWMutex
	games map[string]*game.Game
}

func NewGameStore() *GameStore {
	return &GameStore{
		games: make(map[string]*game.Game),
	}
}

func (s *GameStore) Add(g *game.Game) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.games[g.ID] = g
}

func (s *GameStore) Get(id string) (*game.Game, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.games[id]
	return g, ok
}

func (s *GameStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.games, id)
}
