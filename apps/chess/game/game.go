package game

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
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
	lastActivityNs atomic.Int64           // wall-clock ns of last move; used by solo idle eviction
}

// IsSolo reports whether this is a single-player (non-online) game.
// Solo games have no PlayMeta and are subject to idle-TTL eviction.
func (g *Game) IsSolo() bool { return g.PlayMeta == nil }

// LastActivityNs returns the wall-clock unix-ns of the most recent move
// (or game creation, if no moves have been made).
func (g *Game) LastActivityNs() int64 { return g.lastActivityNs.Load() }

func (g *Game) touchActivity() { g.lastActivityNs.Store(time.Now().UnixNano()) }

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

	g.touchActivity()
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
	g.touchActivity()
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
	g := &Game{
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
	g.touchActivity()
	return g
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

// Resign ends an ongoing game with loser's opponent as winner.
// Returns false if the game was already finished.
func (g *Game) Resign(loser engine.Color) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.State != GameOngoing {
		return false
	}
	g.State = GameResigned
	if loser == engine.White {
		g.Winner = engine.Black
	} else {
		g.Winner = engine.White
	}
	g.signalGameOver()
	return true
}

// AgreeDraw ends an ongoing game in a mutually agreed draw.
// Returns false if the game was already finished.
func (g *Game) AgreeDraw() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.State != GameOngoing {
		return false
	}
	g.State = GameDrawAgreement
	g.Winner = engine.NoColor
	g.signalGameOver()
	return true
}

// IsOngoing returns true if the game is still in progress.
func (g *Game) IsOngoing() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.State == GameOngoing
}

// RevertLastPly drops the most recent half-move from history and rebuilds the
// board from the previous position's FEN. Clocks are NOT refunded — only the
// running side is switched back to the player whose move was reverted. If a
// MoveBatch is attached, the reverted move is purged from persistence (either
// popped from the in-memory buffer or DELETEd from the DB) and a `takeback`
// audit event is appended.
//
// Returns false if there is no move to revert or the game is no longer ongoing.
func (g *Game) RevertLastPly() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != GameOngoing || len(g.History) == 0 {
		return false
	}

	// Side that made the reverted move is the opposite of current side-to-move.
	mover := g.Board.SideToMove ^ 1

	if len(g.History) == 1 {
		g.Board = engine.NewBoard()
	} else {
		prev := g.History[len(g.History)-2]
		b, err := engine.BoardFromFEN(prev.FEN)
		if err != nil {
			return false
		}
		g.Board = b
	}

	revertedSeq := g.Seq

	g.History = g.History[:len(g.History)-1]
	g.legalMoveCache = nil
	g.ClearSelection()
	g.touchActivity()
	if g.Seq > 0 {
		g.Seq--
	}

	// Switch the running clock back to the mover (who is on the move again).
	// Time spent by the opponent thinking on the now-reverted reply is kept
	// (no refund per design choice), but we must NOT route through Clock.Stop
	// because that awards the Fischer increment on completion — takeback is
	// not a move completion. Mutate Running/RemainingNs directly.
	opp := mover ^ 1
	oppClock := &g.Clock.Black
	if opp == engine.White {
		oppClock = &g.Clock.White
	}
	if oppClock.Running {
		elapsed := monoNow() - oppClock.LastStartNs
		if elapsed > 0 {
			oppClock.RemainingNs -= elapsed
			if oppClock.RemainingNs < 0 {
				oppClock.RemainingNs = 0
			}
		}
		oppClock.Running = false
	}
	g.Clock.Start(mover)

	// Purge the reverted move row from persistence (or in-memory buffer) so the
	// next move can reuse the same seq without colliding with the unique
	// (game_id, seq) constraint. Run synchronously while holding g.mu so the
	// next ApplyMove (which also acquires g.mu) cannot race with the cleanup —
	// if it could, the new move could land in the batch buffer with the same
	// seq as the reverted one and RevertLast would pop the wrong entry.
	if g.Batch != nil && revertedSeq > 0 {
		if err := g.Batch.RevertLast(context.Background(), g.ID, revertedSeq); err != nil {
			logger.Error(context.Background()).Err(err).
				Str("game_id", g.ID).Uint64("seq", revertedSeq).
				Msg("RevertLastPly: revert persisted move")
		}
		g.Batch.AppendEvent(PendingEvent{
			GameID:     g.ID,
			SessionID:  g.Batch.sessionID(),
			EventType:  "takeback",
			Payload:    `{"reverted_seq":` + strconv.FormatUint(revertedSeq, 10) + `}`,
			OccurredAt: monoNow(),
		})
	}
	return true
}

// ResumeClockForActiveSide arms the side-to-move clock from its persisted
// remaining time. Used after a server restart when both players have
// reconnected: the game has at least one move on the board, so the clock
// must already have been running for whoever is on the move. We start the
// clock from "now" with the saved remaining time, freezing the downtime
// rather than billing it to the side that was to move.
//
// No-op if the game has no moves yet (the first-move flow takes over),
// is no longer ongoing, or has no timing.
func (g *Game) ResumeClockForActiveSide() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.Timed || g.State != GameOngoing || len(g.History) == 0 {
		return
	}
	g.Clock.Start(g.Board.SideToMove)
}

// StartPlayClock is called by PlayEventsHandler the first time both players
// simultaneously have an SSE connection. It records the first-move deadline
// on PlayMeta. The white clock is NOT started here — it only begins ticking
// after white plays the first move (black's clock starts then).
func (g *Game) StartPlayClock(deadline time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.PlayMeta != nil {
		g.PlayMeta.mu.Lock()
		g.PlayMeta.FirstMoveDeadline = &deadline
		g.PlayMeta.mu.Unlock()
	}
}

// InitBatch wires up DB persistence for this game. Call once after NewGame,
// before the first move is applied. revertFn may be nil for flows that never
// use takeback (e.g. solo games).
func (g *Game) InitBatch(sessionID string, flushFn FlushFunc, gameEndFn GameEndFunc, revertFn MoveRevertFunc) {
	g.Batch = NewMoveBatch(sessionID, flushFn, gameEndFn, revertFn)
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

// MoveResult is the outcome of an attempted move via ApplyMoveChecked.
type MoveResult int

const (
	// MoveApplied: move was legal and is now reflected in the authoritative state.
	MoveApplied MoveResult = iota
	// MoveIllegal: move is not legal in the current position.
	MoveIllegal
	// MoveSeqConflict: caller's baseSeq did not match the game's current Seq —
	// the client is operating on a stale view (typical cause: another move
	// landed in between, or a retried POST after the move already applied).
	MoveSeqConflict
	// MoveGameOver: game is no longer in progress.
	MoveGameOver
	// MovePromoNeeded: move would be a pawn promotion but no promotion piece
	// was specified — caller must re-submit with a Promotion field set.
	MovePromoNeeded
)

func (r MoveResult) String() string {
	switch r {
	case MoveApplied:
		return "applied"
	case MoveIllegal:
		return "illegal"
	case MoveSeqConflict:
		return "seq-conflict"
	case MoveGameOver:
		return "game-over"
	case MovePromoNeeded:
		return "promo-needed"
	}
	return "unknown"
}

// SignalsSnapshot is an atomic view of the game state needed to build a
// ChessBoardSignals patch. Captured under g.mu so two near-simultaneous
// moves cannot interleave between ApplyMoveChecked returning and the caller
// reading g.Board/g.Seq/g.State to build the broadcast — which previously
// allowed a hub frame to carry seq=N+1 paired with the FEN from seq=N.
type SignalsSnapshot struct {
	Timed            bool
	SideToMove       engine.Color
	Fen              string
	Seq              uint64
	IsCheck          bool
	State            GameState
	Winner           engine.Color
	HasSelection     bool
	SelectionFrom    uint8
	SelectionTargets []uint8
}

// SignalsSnapshot takes a read lock and returns an atomic view suitable for
// building a ChessBoardSignals patch. Use this when constructing a broadcast
// outside of ApplyMoveChecked (which returns its own snapshot under the
// write lock).
func (g *Game) ReadSignalsSnapshot() SignalsSnapshot {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.signalsSnapshotLocked()
}

// signalsSnapshotLocked must be called with g.mu held (read or write).
func (g *Game) signalsSnapshotLocked() SignalsSnapshot {
	snap := SignalsSnapshot{
		Timed:      g.Timed,
		SideToMove: g.Board.SideToMove,
		Fen:        g.Board.FEN(),
		Seq:        g.Seq,
		IsCheck:    g.Board.IsKingInCheck(g.Board.SideToMove),
		State:      g.State,
		Winner:     g.Winner,
	}
	if g.Selection != nil {
		snap.HasSelection = true
		snap.SelectionFrom = g.Selection.FromSquare
		snap.SelectionTargets = append([]uint8(nil), g.Selection.Targets...)
	}
	return snap
}

// ApplyMoveChecked validates and applies a move under a single lock, with an
// optional baseSeq guard so two near-simultaneous requests from the same
// client cannot land out of order. baseSeq < 0 disables the check (solo /
// non-online callers). Returns the outcome and a snapshot of the resulting
// game state — captured under the same lock as the apply, so the caller can
// build an atomically-consistent broadcast without re-reading g.
func (g *Game) ApplyMoveChecked(m engine.Move, lagCompNs int64, baseSeq int64) (MoveResult, SignalsSnapshot) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != GameOngoing {
		return MoveGameOver, g.signalsSnapshotLocked()
	}
	if baseSeq >= 0 && uint64(baseSeq) != g.Seq {
		return MoveSeqConflict, g.signalsSnapshotLocked()
	}
	// Validate move legality. Board.MakeMove only checks king-safety, not
	// piece-movement geometry (sliders ignore blockers), so a misbehaving or
	// malicious client could otherwise teleport pieces. Use the move generator
	// to confirm (from→to[,promo]) is in the legal-move set for the side to
	// move. Pre-Step-2 this was enforced by the server-side selection state
	// (SelectSquare→IsTarget) — that path is gone, so the check moves here.
	legal := g.Board.GenerateMovesForSquare(m.From)
	matched := false
	needsPromo := false
	for _, lm := range legal {
		if lm.To != m.To {
			continue
		}
		// Check the MovePromo flag rather than the Promotion field — the
		// generator leaves Promotion as its zero value (Pawn) for non-promo
		// moves, so comparing against NoPiece (255) is unreliable.
		if lm.Flags&engine.MovePromo != 0 {
			if m.Promotion == engine.NoPiece {
				needsPromo = true
				continue
			}
			if lm.Promotion == m.Promotion {
				matched = true
				break
			}
		} else {
			matched = true
			break
		}
	}
	if !matched {
		if needsPromo {
			return MovePromoNeeded, g.signalsSnapshotLocked()
		}
		return MoveIllegal, g.signalsSnapshotLocked()
	}
	if !g.applyMoveLocked(m, lagCompNs) {
		return MoveIllegal, g.signalsSnapshotLocked()
	}
	return MoveApplied, g.signalsSnapshotLocked()
}

func (g *Game) ApplyMove(m engine.Move, lagCompNs int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.State != GameOngoing {
		return false
	}
	return g.applyMoveLocked(m, lagCompNs)
}

// applyMoveLocked is the core move-apply routine; the caller must hold g.mu
// and must have already verified g.State == GameOngoing.
func (g *Game) applyMoveLocked(m engine.Move, lagCompNs int64) bool {
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
	g.touchActivity()

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
	if g.State == GameOngoing {
		nowNs := monoNow()
		signals := map[string]any{}
		if g.Timed {
			signals["clkW"] = g.Clock.RemainingAt(0, nowNs)
			signals["clkB"] = g.Clock.RemainingAt(1, nowNs)
			signals["clkWRun"] = g.Clock.White.Running
			signals["clkBRun"] = g.Clock.Black.Running
			signals["clkTs"] = nowNs
		}
		// First move in a play game: clear the pre-game countdown signal.
		if g.PlayMeta != nil && len(g.History) == 1 {
			signals["firstMoveDeadlineNs"] = 0
		}
		if len(signals) > 0 {
			go g.Hub.BroadcastSignals(signals)
		}
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
	signals := map[string]any{
		"gameState":     int(g.State),
		"gameStateText": stateText,
		"winner":        int(g.Winner),
		"claimVictory":  false,
	}
	go g.Hub.BroadcastSignals(signals)
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
	return g.isPromotionMoveLocked(move)
}

// isPromotionMoveLocked is the unlocked core of IsPromotionMove. Caller must
// hold g.mu (read or write). Used by ApplyMoveChecked to detect a missing
// promotion piece without releasing and reacquiring the lock.
func (g *Game) isPromotionMoveLocked(move engine.Move) bool {
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
