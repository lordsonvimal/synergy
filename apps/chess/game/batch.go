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

// MoveRevertFunc deletes persisted moves with seq >= fromSeq for a given game.
// Used by takeback to purge reverted rows from the DB.
type MoveRevertFunc func(ctx context.Context, gameID string, fromSeq uint64) error

// MoveBatch accumulates moves and events in memory and flushes them to the
// DB either periodically, when the batch size threshold is hit, or on game end.
type MoveBatch struct {
	mu        sync.Mutex
	flushMu   sync.Mutex // serializes DB writes so revert can wait for in-flight flush
	moves     []PendingMove
	events    []PendingEvent
	flushFn   FlushFunc
	gameEndFn GameEndFunc
	revertFn  MoveRevertFunc
	sid       string // session ID, embedded in every row
	ticker    *time.Ticker
	stopCh    chan struct{}
}

func (b *MoveBatch) sessionID() string { return b.sid }

// NewMoveBatch creates a batch and starts the background flush ticker.
// revertFn may be nil for solo games that never use takeback.
func NewMoveBatch(sessionID string, flushFn FlushFunc, gameEndFn GameEndFunc, revertFn MoveRevertFunc) *MoveBatch {
	b := &MoveBatch{
		moves:     make([]PendingMove, 0, batchFlushSize),
		events:    make([]PendingEvent, 0, batchFlushSize),
		flushFn:   flushFn,
		gameEndFn: gameEndFn,
		revertFn:  revertFn,
		sid:       sessionID,
		ticker:    time.NewTicker(batchFlushInterval),
		stopCh:    make(chan struct{}),
	}
	go b.loop()
	return b
}

// RevertLast removes a single move (by seq) from persistence. If the move is
// still buffered in memory it is popped; otherwise the row is DELETEd from the
// DB. Any in-flight flush is awaited (via flushMu) so we cannot lose a delete
// against a concurrent insert of the same seq.
//
// Returns nil if revertFn is nil and no in-memory match was found (solo flow).
func (b *MoveBatch) RevertLast(ctx context.Context, gameID string, seq uint64) error {
	b.mu.Lock()
	if n := len(b.moves); n > 0 && b.moves[n-1].Seq == seq && b.moves[n-1].GameID == gameID {
		b.moves = b.moves[:n-1]
		if m := len(b.events); m > 0 {
			b.events = b.events[:m-1]
		}
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	// Move already drained out of the buffer — wait for any in-flight DB write
	// to finish, then issue the delete so we don't race with the insert.
	b.flushMu.Lock()
	b.flushMu.Unlock()
	if b.revertFn == nil {
		return nil
	}
	return b.revertFn(ctx, gameID, seq)
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

// AppendEvent adds a standalone audit event (no associated move) and schedules
// a flush. Used for non-move lifecycle entries like "takeback".
func (b *MoveBatch) AppendEvent(event PendingEvent) {
	b.mu.Lock()
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

// FlushNow drains any pending moves/events AND waits for any concurrent
// in-flight flush to complete before returning. Use this from the move POST
// path before acknowledging the move to the player: without it the response
// can race the async DB write, so a server crash in that window would lose
// a move the player has already seen.
//
// flush() takes an early-return shortcut when its drain finds the buffer
// empty (because a concurrent Append-triggered goroutine drained it first).
// The unconditional flushMu Lock/Unlock here is what gives us the "wait for
// the other writer" guarantee in that case.
func (b *MoveBatch) FlushNow(ctx context.Context) {
	b.flush(ctx)
	b.flushMu.Lock()
	b.flushMu.Unlock()
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

	// Serialize the actual DB write so RevertLast (or another flush) cannot
	// interleave with this one. RevertLast briefly grabs flushMu after
	// draining the in-memory buffer to ensure pending writes complete before
	// it issues a DELETE.
	b.flushMu.Lock()
	defer b.flushMu.Unlock()
	b.flushFn(ctx, moves, events)
}
