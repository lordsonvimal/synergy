package game

import (
	"context"
	"sync"
	"time"
)

const (
	batchFlushSize     = 20
	batchFlushInterval = 30 * time.Second
)

// PendingMove holds data for one played move awaiting DB persistence.
type PendingMove struct {
	GameID      string
	SessionID   string
	Seq         uint64
	UCI         string
	SAN         string
	FEN         string
	MoveNumber  int
	Color       string // "white" | "black"
	WRemNs      int64
	BRemNs      int64
	LagCompNs   int64
	ThinkTimeNs int64
	PlayedAt    int64
}

// PendingEvent holds a game lifecycle event awaiting DB persistence.
type PendingEvent struct {
	GameID     string
	SessionID  string
	EventType  string
	Payload    string
	OccurredAt int64
}

// FlushFunc persists a batch of moves and events in one DB transaction.
type FlushFunc func(ctx context.Context, moves []PendingMove, events []PendingEvent) error

// GameEndFunc updates the game row when a game concludes.
type GameEndFunc func(ctx context.Context, status, winner string) error

// MoveBatch accumulates moves and events in memory and flushes them to the
// DB either periodically, when the batch size threshold is hit, or on game end.
type MoveBatch struct {
	mu        sync.Mutex
	moves     []PendingMove
	events    []PendingEvent
	flushFn   FlushFunc
	gameEndFn GameEndFunc
	sid       string // session ID, embedded in every row
	ticker    *time.Ticker
	stopCh    chan struct{}
}

func (b *MoveBatch) sessionID() string { return b.sid }

// NewMoveBatch creates a batch and starts the background flush ticker.
func NewMoveBatch(sessionID string, flushFn FlushFunc, gameEndFn GameEndFunc) *MoveBatch {
	b := &MoveBatch{
		moves:     make([]PendingMove, 0, batchFlushSize),
		events:    make([]PendingEvent, 0, batchFlushSize),
		flushFn:   flushFn,
		gameEndFn: gameEndFn,
		sid:       sessionID,
		ticker:    time.NewTicker(batchFlushInterval),
		stopCh:    make(chan struct{}),
	}
	go b.loop()
	return b
}

func (b *MoveBatch) loop() {
	for {
		select {
		case <-b.stopCh:
			return
		case <-b.ticker.C:
			b.flush(context.Background())
		}
	}
}

// Append adds a move (and its corresponding event) and immediately schedules
// an async flush so every move reaches the DB without waiting for the ticker.
func (b *MoveBatch) Append(move PendingMove, event PendingEvent) {
	b.mu.Lock()
	b.moves = append(b.moves, move)
	b.events = append(b.events, event)
	b.mu.Unlock()
	go b.flush(context.Background())
}

// FlushAndStop performs a final flush and stops the background ticker.
// It should be called once when the game ends, after setting state/winner.
func (b *MoveBatch) FlushAndStop(ctx context.Context, status, winner string) {
	select {
	case <-b.stopCh:
		return
	default:
		close(b.stopCh)
	}
	b.ticker.Stop()
	b.flush(ctx)
	if b.gameEndFn != nil {
		b.gameEndFn(ctx, status, winner)
	}
}

// flush drains the pending moves and events and writes them to DB.
func (b *MoveBatch) flush(ctx context.Context) {
	b.mu.Lock()
	if len(b.moves) == 0 && len(b.events) == 0 {
		b.mu.Unlock()
		return
	}
	moves := make([]PendingMove, len(b.moves))
	events := make([]PendingEvent, len(b.events))
	copy(moves, b.moves)
	copy(events, b.events)
	b.moves = b.moves[:0]
	b.events = b.events[:0]
	b.mu.Unlock()

	b.flushFn(ctx, moves, events)
}
