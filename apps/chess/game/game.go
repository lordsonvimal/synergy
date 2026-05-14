package game

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lordsonvimal/synergy/apps/chess/engine"
	"github.com/lordsonvimal/synergy/apps/chess/logger"
)

type SelectionState struct {
	FromSquare uint8
	Targets    []uint8
}

type SelectionSnapshot struct {
	FromSquare int   `json:"selectedSquare"`
	Targets    []int `json:"possibleMoves"`
}

type GameState int

const (
	GameOngoing                  GameState = iota
	GameCheckmate                          // checkmate
	GameResigned                           // player resigned
	GameClockFlagged                       // player ran out of time
	GameDrawStalemate                      // stalemate
	GameDrawFiftyMove                      // fifty-move rule
	GameDrawAgreement                      // mutual agreement
	GameDrawThreefoldRepetition            // threefold repetition
	GameDrawInsufficientMaterial           // insufficient material
	GameAbandoned                          // player left the game
	GameDisconnected                       // network error causing game to stop
	GameInvalid                            // invalid state
)

type Game struct {
	ID            string
	Board         *engine.Board
	Clock         GameClock
	Batch         *MoveBatch
	Hub           *GameHub
	Selection     *SelectionState
	History       []MoveRecord
	Seq           uint64
	State         GameState
	Winner        engine.Color // valid after game over
	PlayMeta      *PlayMeta    // nil for /game/* solo flow
	Timed         bool         // false = no clock, watchdog skips flag checks
	InitialTimeNs int64        // starting time control in ns; 0 for untimed games

	mu             sync.RWMutex
	legalMoveCache map[engine.Color]bool // cache per side
	stopCh         chan struct{}          // closed to stop the watchdog
}

func NewGame(mode *GameMode) *Game {
	board := engine.NewBoard()
	id := uuid.New().String()
	gc := NewClock(mode.TimeNs, mode.Increment)
	gc.Start(engine.White)

	g := &Game{
		ID:            id,
		Board:         board,
		Clock:         gc,
		Hub:           NewGameHub(),
		History:       make([]MoveRecord, 0),
		Seq:           0,
		State:         GameOngoing,
		Winner:        engine.NoColor,
		Timed:         mode.Timed,
		InitialTimeNs: mode.TimeNs,
		stopCh:        make(chan struct{}),
	}

	if mode.Timed {
		g.startWatchdog()
	}
	return g
}

// NewPlayGame creates an online two-player game. The clock is NOT started at
// creation — it starts when both players first connect via SSE.
func NewPlayGame(mode *GameMode, meta *PlayMeta) *Game {
	board := engine.NewBoard()
	id := uuid.New().String()
	gc := NewClock(mode.TimeNs, mode.Increment)

	g := &Game{
		ID:            id,
		Board:         board,
		Clock:         gc,
		Hub:           NewGameHub(),
		History:       make([]MoveRecord, 0),
		Seq:           0,
		State:         GameOngoing,
		Winner:        engine.NoColor,
		PlayMeta:      meta,
		Timed:         mode.Timed,
		InitialTimeNs: mode.TimeNs,
		stopCh:        make(chan struct{}),
	}
	g.startWatchdog()
	return g
}

// NewRestoredPlayGame builds an ongoing play game from DB after a server restart.
// The clock is not started; it waits for both players to reconnect via SSE.
func NewRestoredPlayGame(
	id string, board *engine.Board, clock GameClock,
	history []MoveRecord, seq uint64, state GameState, winner engine.Color,
	meta *PlayMeta,
) *Game {
	stopCh := make(chan struct{})
	if state != GameOngoing {
		close(stopCh)
	}
	g := &Game{
		ID:            id,
		Board:         board,
		Clock:         clock,
		Hub:           NewGameHub(),
		History:       history,
		Seq:           seq,
		State:         state,
		Winner:        winner,
		PlayMeta:      meta,
		Timed:         meta.OriginalMode.Timed,
		InitialTimeNs: meta.OriginalMode.TimeNs,
		stopCh:        stopCh,
	}
	if state == GameOngoing {
		g.startWatchdog()
	}
	return g
}

// NewRestoredGame builds a Game from fields loaded out of the DB.
// Unexported fields are initialised safely; no watchdog is started.
func NewRestoredGame(
	id string, board *engine.Board, clock GameClock,
	history []MoveRecord, seq uint64, state GameState, winner engine.Color, timed bool, initialTimeNs int64,
) *Game {
	stopCh := make(chan struct{})
	if state != GameOngoing {
		close(stopCh)
	}
	return &Game{
		ID:            id,
		Board:         board,
		Clock:         clock,
		Hub:           NewGameHub(),
		History:       history,
		Seq:           seq,
		State:         state,
		Winner:        winner,
		Timed:         timed,
		InitialTimeNs: initialTimeNs,
		stopCh:        stopCh,
	}
}

// BroadcastEvent marshals v as JSON and broadcasts it to all SSE subscribers.
func (g *Game) BroadcastEvent(v any) {
	go g.broadcastJSON(v)
}

// AbandonWithResult sets the game as abandoned, optionally with a winner,
// and signals game over. Returns false if the game was already finished.
func (g *Game) AbandonWithResult(winner engine.Color) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.State != GameOngoing {
		return false
	}
	g.State = GameAbandoned
	g.Winner = winner
	g.signalGameOver()
	return true
}

// IsOngoing returns true if the game is still in progress.
func (g *Game) IsOngoing() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.State == GameOngoing
}

// StartPlayClock is called by PlayEventsHandler the first time both players
// simultaneously have an SSE connection. It records the first-move deadline
// on PlayMeta and starts the white clock.
func (g *Game) StartPlayClock(deadline time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.PlayMeta != nil {
		g.PlayMeta.mu.Lock()
		g.PlayMeta.FirstMoveDeadline = &deadline
		g.PlayMeta.mu.Unlock()
	}
	g.Clock.Start(engine.White)
}

// InitBatch wires up DB persistence for this game. Call once after NewGame,
// before the first move is applied.
func (g *Game) InitBatch(sessionID string, flushFn FlushFunc, gameEndFn GameEndFunc) {
	g.Batch = NewMoveBatch(sessionID, flushFn, gameEndFn)
}

// --------------------------
// Check if current side's king is in check
// --------------------------
func (g *Game) IsCheck() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Board.IsKingInCheck(g.Board.SideToMove)
}

// --------------------------
// Check if current side is checkmated
// --------------------------
func (g *Game) IsCheckmate() bool {
	color := g.Board.SideToMove

	// If king is not in check, cannot be checkmate
	if !g.Board.IsKingInCheck(color) {
		return false
	}

	// Check legal moves lazily
	return !g.hasLegalMoves(color)
}

// --------------------------
// Check if current side is stalemated
// --------------------------
func (g *Game) IsStalemate() bool {
	// g.mu.RLock()
	// defer g.mu.RUnlock()

	color := g.Board.SideToMove

	// If king is in check, cannot be stalemate
	if g.Board.IsKingInCheck(color) {
		return false
	}

	// Check legal moves lazily
	return !g.hasLegalMoves(color)
}

// --------------------------
// Apply move
// --------------------------
func (g *Game) ApplyMove(m engine.Move, lagCompNs int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 0. Prevent moves if game is already over
	if g.State != GameOngoing {
		return false
	}

	color := g.Board.SideToMove
	// Capture SAN and move metadata BEFORE the move so the board is still in the
	// pre-move position. If MakeMove subsequently fails, history is not appended.
	san := g.Board.SAN(m)
	moveNumber := int(g.Board.FullMoveNumber)

	// Capture remaining times before Stop() so we can compute think_time_ns.
	prevWRem := g.Clock.White.RemainingNs
	prevBRem := g.Clock.Black.RemainingNs

	g.Clock.Stop(color, lagCompNs)

	if !g.Board.MakeMove(m) {
		return false
	}

	g.History = append(g.History, MoveRecord{
		SAN:        san,
		FEN:        g.Board.FEN(),
		Color:      color,
		MoveNumber: moveNumber,
		WRemNs:     g.Clock.White.RemainingNs,
		BRemNs:     g.Clock.Black.RemainingNs,
	})

	// Reset legal move cache since board changed
	g.legalMoveCache = nil

	g.Clock.Start(color ^ 1)

	g.Seq++
	if g.Batch != nil {
		now := monoNow()
		colorStr := "white"
		if color == engine.Black {
			colorStr = "black"
		}
		var thinkTimeNs int64
		if color == engine.White {
			thinkTimeNs = prevWRem - g.Clock.White.RemainingNs + g.Clock.IncNs
		} else {
			thinkTimeNs = prevBRem - g.Clock.Black.RemainingNs + g.Clock.IncNs
		}
		uci := m.ToUCI()
		g.Batch.Append(
			PendingMove{
				GameID:      g.ID,
				SessionID:   g.Batch.sessionID(),
				Seq:         g.Seq,
				UCI:         uci,
				SAN:         san,
				FEN:         g.Board.FEN(),
				MoveNumber:  moveNumber,
				Color:       colorStr,
				WRemNs:      g.Clock.White.RemainingNs,
				BRemNs:      g.Clock.Black.RemainingNs,
				LagCompNs:   lagCompNs,
				ThinkTimeNs: thinkTimeNs,
				PlayedAt:    now,
			},
			PendingEvent{
				GameID:     g.ID,
				SessionID:  g.Batch.sessionID(),
				EventType:  "move_played",
				Payload:    `{"uci":"` + uci + `","san":"` + san + `"}`,
				OccurredAt: now,
			},
		)
	}

	g.ClearSelection()  // After move, clear selection
	g.UpdateGameState() // Update game state after each move

	// Push an immediate clock tick so the client switches sides without
	// waiting up to 1s for the next watchdog broadcast.
	if g.Timed && g.State == GameOngoing {
		nowNs := monoNow()
		evt := ClockTickEvent{
			Type:             "clock_tick",
			WhiteRemainingNs: g.Clock.RemainingAt(0, nowNs),
			BlackRemainingNs: g.Clock.RemainingAt(1, nowNs),
			WhiteRunning:     g.Clock.White.Running,
			BlackRunning:     g.Clock.Black.Running,
			ServerTsNs:       nowNs,
		}
		go g.broadcastJSON(evt)
	}

	return true
}

// --------------------------
// Update game state after a move
// --------------------------
func (g *Game) UpdateGameState() {
	// Skip if game already over
	if g.State != GameOngoing {
		return
	}

	color := g.Board.SideToMove ^ 1 // The player who just moved

	// 1. Check checkmate
	if g.IsCheckmate() {
		g.State = GameCheckmate
		g.Winner = color
		g.signalGameOver()
		return
	}

	// 2. Check stalemate
	if g.IsStalemate() {
		g.State = GameDrawStalemate
		g.Winner = engine.NoColor
		g.signalGameOver()
		return
	}

	// 3. Threefold repetition
	if g.Board.IsThreefoldRepetition() {
		g.State = GameDrawThreefoldRepetition
		g.Winner = engine.NoColor
		g.signalGameOver()
		return
	}

	// 4. Insufficient material
	if g.Board.IsInsufficientMaterial() {
		g.State = GameDrawInsufficientMaterial
		g.Winner = engine.NoColor
		g.signalGameOver()
		return
	}

	g.State = GameOngoing
	g.Winner = engine.NoColor
}

// signalGameOver stops the watchdog and broadcasts a game_over SSE event.
// Must be called after g.State and g.Winner have been set.
// Called with the game mutex held — broadcasts are async so no deadlock.
func (g *Game) signalGameOver() {
	// Freeze both clocks so no side keeps counting down after the game ends.
	g.Clock.White.Running = false
	g.Clock.Black.Running = false

	select {
	case <-g.stopCh:
		// already stopped
	default:
		close(g.stopCh)
	}

	if g.Batch != nil {
		status := gameStateDBStatus(g.State)
		winner := ""
		if g.Winner == engine.White {
			winner = "white"
		} else if g.Winner == engine.Black {
			winner = "black"
		}
		go g.Batch.FlushAndStop(context.Background(), status, winner)
	}

	stateText := gameStateText(g.State)
	evt := GameOverEvent{
		Type:      "game_over",
		State:     g.State,
		Winner:    int(g.Winner),
		StateText: stateText,
	}
	go g.broadcastJSON(evt)
}

func gameStateDBStatus(s GameState) string {
	switch s {
	case GameCheckmate:
		return "checkmate"
	case GameResigned:
		return "resigned"
	case GameClockFlagged:
		return "clock_flagged"
	case GameDrawStalemate:
		return "draw_stalemate"
	case GameDrawFiftyMove:
		return "draw_fifty_move"
	case GameDrawAgreement:
		return "draw_agreement"
	case GameDrawThreefoldRepetition:
		return "draw_threefold"
	case GameDrawInsufficientMaterial:
		return "draw_insufficient"
	case GameAbandoned:
		return "abandoned"
	case GameDisconnected:
		return "disconnected"
	default:
		return "abandoned"
	}
}

func gameStateText(s GameState) string {
	switch s {
	case GameCheckmate:
		return "Checkmate"
	case GameResigned:
		return "Resigned"
	case GameClockFlagged:
		return "Clock flagged"
	case GameDrawStalemate:
		return "Stalemate"
	case GameDrawFiftyMove:
		return "Fifty-move rule"
	case GameDrawAgreement:
		return "Draw by agreement"
	case GameDrawThreefoldRepetition:
		return "Threefold repetition"
	case GameDrawInsufficientMaterial:
		return "Insufficient material"
	case GameAbandoned:
		return "Abandoned"
	case GameDisconnected:
		return "Disconnected"
	default:
		return "Unknown"
	}
}

// --------------------------
// Helper: cached HasLegalMoves
// --------------------------
func (g *Game) hasLegalMoves(color engine.Color) bool {
	if g.legalMoveCache == nil {
		g.legalMoveCache = make(map[engine.Color]bool)
	}
	if val, ok := g.legalMoveCache[color]; ok {
		return val
	}
	val := g.Board.HasLegalMoves(color)
	g.legalMoveCache[color] = val
	return val
}

// HasSelection returns true if a square is currently selected for the next move.
func (g *Game) HasSelection() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Selection != nil
}

// GetSelectionFrom returns the currently selected square index.
// Note: You should check HasSelection() before calling this.
func (g *Game) GetSelectionFrom() uint8 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Selection == nil {
		return 255 // Return invalid square if no selection
	}
	return g.Selection.FromSquare
}

func (g *Game) SelectionSnapshot() SelectionSnapshot {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.Selection == nil {
		return SelectionSnapshot{
			FromSquare: 255, // No selection
			Targets:    []int{},
		}
	}

	moves := make([]int, len(g.Selection.Targets))
	for i, t := range g.Selection.Targets {
		moves[i] = int(t)
	}

	return SelectionSnapshot{
		FromSquare: int(g.Selection.FromSquare),
		Targets:    moves,
	}
}

// IsTarget checks if the provided square is a valid move target for the current selection.
func (g *Game) IsTarget(square uint8) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Selection == nil {
		return false
	}
	for _, t := range g.Selection.Targets {
		if t == square {
			return true
		}
	}
	return false
}

func (g *Game) IsPromotionMove(move engine.Move) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	color, piece, _ := g.Board.PieceAt(move.From)
	if piece != engine.Pawn {
		return false
	}

	if !g.Board.TryMove(move) {
		return false
	}

	rank := move.To / 8
	if color == engine.White {
		return rank == 7
	}

	return rank == 0
}

func (g *Game) ClearSelection() {
	g.Selection = nil
}

// HistoryFENAt returns the FEN string for the position after the given
// zero-based half-move index. The second return value is false if idx is
// out of range.
func (g *Game) HistoryFENAt(idx int) (string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if idx < 0 || idx >= len(g.History) {
		return "", false
	}
	return g.History[idx].FEN, true
}

// HistoryAt returns the full MoveRecord at the given zero-based half-move index,
// including clock remaining times. The second return value is false if out of range.
func (g *Game) HistoryAt(idx int) (MoveRecord, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if idx < 0 || idx >= len(g.History) {
		return MoveRecord{}, false
	}
	return g.History[idx], true
}

// ClockTickSnapshot returns the current clock state as a ClockTickEvent.
func (g *Game) ClockTickSnapshot() ClockTickEvent {
	g.mu.RLock()
	defer g.mu.RUnlock()
	nowNs := time.Now().UnixNano()
	return ClockTickEvent{
		Type:             "clock_tick",
		WhiteRemainingNs: g.Clock.RemainingAt(0, nowNs),
		BlackRemainingNs: g.Clock.RemainingAt(1, nowNs),
		WhiteRunning:     g.Clock.White.Running,
		BlackRunning:     g.Clock.Black.Running,
		ServerTsNs:       nowNs,
	}
}

// HistoryLen returns the number of half-moves recorded so far.
func (g *Game) HistoryLen() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.History)
}

func (g *Game) SelectSquare(ctx context.Context, square uint8) {
	g.mu.Lock()

	color, _, ok := g.Board.PieceAt(square)
	// If no piece or piece is not ours, clear selection
	if !ok || color != g.Board.SideToMove {
		g.Selection = nil
		g.mu.Unlock()

		logger.Info(ctx).Msg("Invalid piece")
		return
	}

	// Generate moves and update selection
	// Assuming you implement GenerateMovesForSquare in your engine
	moves := g.Board.GenerateMovesForSquare(square)

	targets := make([]uint8, 0, len(moves))
	for _, m := range moves {
		targets = append(targets, m.To)
	}

	g.Selection = &SelectionState{
		FromSquare: square,
		Targets:    targets,
	}

	g.mu.Unlock()

	logger.Info(ctx).
		Int("target lengths", len(moves)).
		Msg("SelectSquare EXIT: selected")
}
