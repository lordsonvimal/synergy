package engine

import (
	"sync"
	"sync/atomic"
)

type TTEntryType int8

const (
	TTExact TTEntryType = iota
	TTLowerBound
	TTUpperBound
)

type TTEntry struct {
	Hash     uint64
	Depth    int
	Value    int
	Type     TTEntryType
	BestMove Move
	Age      uint8
}

type TranspositionTable struct {
	table    map[uint64]TTEntry
	capacity int
	mu       sync.RWMutex
	age      uint8

	Hits   atomic.Uint64
	Misses atomic.Uint64
}

func NewTT(capacity int) *TranspositionTable {
	return &TranspositionTable{
		table:    make(map[uint64]TTEntry, capacity),
		capacity: capacity,
	}
}

func (tt *TranspositionTable) Probe(
	hash uint64,
	depth int,
	alpha int,
	beta int,
) (int, Move, bool) {

	tt.mu.RLock()
	entry, ok := tt.table[hash]
	tt.mu.RUnlock()

	if !ok {
		tt.Misses.Add(1)
		return 0, Move{}, false
	}

	tt.Hits.Add(1)

	if entry.Depth < depth {
		return 0, Move{}, false
	}

	switch entry.Type {
	case TTExact:
		return entry.Value, entry.BestMove, true
	case TTLowerBound:
		if entry.Value >= beta {
			return entry.Value, entry.BestMove, true
		}
	case TTUpperBound:
		if entry.Value <= alpha {
			return entry.Value, entry.BestMove, true
		}
	}

	return denormalizeScore(entry.Value, depth), entry.BestMove, true
}

func normalizeScore(score, ply int) int {
	const mate = 100000
	if score > mate-1000 {
		return score + ply
	}
	if score < -mate+1000 {
		return score - ply
	}
	return score
}

func denormalizeScore(score, ply int) int {
	const mate = 100000
	if score > mate-1000 {
		return score - ply
	}
	if score < -mate+1000 {
		return score + ply
	}
	return score
}

func (tt *TranspositionTable) Store(
	hash uint64,
	depth int,
	value int,
	entryType TTEntryType,
	best Move,
	ply int,
) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	value = normalizeScore(value, ply)

	if old, ok := tt.table[hash]; ok {
		if old.Age == tt.age && old.Depth > depth {
			return
		}
	}

	if len(tt.table) >= tt.capacity {
		tt.evict()
	}

	tt.table[hash] = TTEntry{
		Hash:     hash,
		Depth:    depth,
		Value:    value,
		Type:     entryType,
		BestMove: best,
		Age:      tt.age,
	}
}

func (tt *TranspositionTable) NewSearch() {
	tt.mu.Lock()
	tt.age++
	tt.mu.Unlock()
}

func (tt *TranspositionTable) evict() {
	for k, v := range tt.table {
		if v.Age != tt.age {
			delete(tt.table, k)
			return
		}
	}
	// fallback
	for k := range tt.table {
		delete(tt.table, k)
		return
	}
}
