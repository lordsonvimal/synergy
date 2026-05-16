package store

import (
	"sync"
	"time"

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

// soloIdleTTL is how long a solo / computer game can sit untouched in memory
// before the sweeper evicts it. Solo games are not persisted, so eviction
// just drops the in-memory state — the user gets "game not found" if they
// come back later.
const soloIdleTTL = time.Hour

// soloSweepInterval is how often the sweeper scans for stale solo games.
// 5 minutes keeps the sweep cheap on a multi-thousand-game map while still
// freeing memory reasonably soon after the TTL elapses.
const soloSweepInterval = 5 * time.Minute

type GameStore struct {
	mu    sync.RWMutex
	games map[string]*game.Game
}

func NewGameStore() *GameStore {
	s := &GameStore{
		games: make(map[string]*game.Game),
	}
	go s.sweepLoop()
	return s
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

// sweepLoop periodically evicts solo games whose last activity is older than
// soloIdleTTL. Play games are never touched here — they have their own
// lifecycle (abandonment, rematch, etc.) and are recoverable via the DB.
func (s *GameStore) sweepLoop() {
	t := time.NewTicker(soloSweepInterval)
	defer t.Stop()
	for range t.C {
		s.sweepOnce(time.Now().UnixNano())
	}
}

func (s *GameStore) sweepOnce(nowNs int64) {
	cutoff := nowNs - soloIdleTTL.Nanoseconds()
	var stale []string
	s.mu.RLock()
	for id, g := range s.games {
		if g.IsSolo() && g.LastActivityNs() < cutoff {
			stale = append(stale, id)
		}
	}
	s.mu.RUnlock()
	if len(stale) == 0 {
		return
	}
	s.mu.Lock()
	for _, id := range stale {
		// Re-check under the write lock in case the game was touched (or
		// already deleted) between the scan and the delete.
		if g, ok := s.games[id]; ok && g.IsSolo() && g.LastActivityNs() < cutoff {
			delete(s.games, id)
		}
	}
	s.mu.Unlock()
}
