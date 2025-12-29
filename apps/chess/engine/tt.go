package engine

import "sync"

type TTEntryType int

const (
	Exact TTEntryType = iota
	LowerBound
	UpperBound
)

type TTEntry struct {
	Hash      uint64
	Depth     int
	Value     int
	EntryType TTEntryType
	BestMove  Move
}

type TranspositionTable struct {
	table map[uint64]TTEntry
	mu    sync.RWMutex
}

func NewTT() *TranspositionTable {
	return &TranspositionTable{
		table: make(map[uint64]TTEntry),
	}
}

// Store entry
func (tt *TranspositionTable) Store(hash uint64, depth int, value int, et TTEntryType, best Move) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	tt.table[hash] = TTEntry{
		Hash:      hash,
		Depth:     depth,
		Value:     value,
		EntryType: et,
		BestMove:  best,
	}
}

// Lookup entry
func (tt *TranspositionTable) Lookup(hash uint64) (TTEntry, bool) {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	entry, ok := tt.table[hash]
	return entry, ok
}
