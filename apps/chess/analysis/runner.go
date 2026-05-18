// Package analysis runs background engine analyses keyed by game ID. One job
// per game is in flight at a time; starting a new one cancels the previous.
//
// The Runner sits on top of engine/uci.Pool and converts UCI Updates into
// display-ready EvalSnapshots (eval from white's POV, win%, depth).
package analysis

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/lordsonvimal/synergy/apps/chess/engine/uci"
)

// histKey returns the cancellation slot key used for history navigation jobs.
// Kept centralised so the broadcast (Start), cleanup (Forget), and explicit
// cancel (CancelHistory) paths can't drift apart.
func histKey(gameID string) string { return "hist:" + gameID }

// jobSlot pairs a cancel func with a monotonic id so the run() defer can tell
// whether the slot it installed is still current. Comparing CancelFunc values
// directly does not work — they're closures sharing a code pointer, so
// reflect.Value.Pointer() returns the same value for every cancel produced by
// context.WithTimeout and can't distinguish instances.
type jobSlot struct {
	id     uint64
	cancel context.CancelFunc
	// done closes when the slot's goroutine has finished its defer. Used by
	// CancelHistory to wait out any straggler emission that was already in
	// flight when cancellation arrived — otherwise that last frame would
	// reach subscribers via the hub *after* the return-to-live direct push,
	// snapping the eval bar back to the historical position.
	done chan struct{}
}

// EvalSnapshot is what the eval bar UI binds to. Sent as a DataStar signal patch.
type EvalSnapshot struct {
	EvalCp          int     `json:"evalCp"`          // centipawns from white's POV
	EvalMate        int     `json:"evalMate"`        // signed mate distance from white's POV; 0 if none
	EvalDepth       int     `json:"evalDepth"`       // search depth in plies
	EvalWp          float64 `json:"evalWp"`          // win% from white's POV, 0..100
	EvalDisplay     string  `json:"evalDisplay"`     // e.g. "+0.42" or "M5"
	EvalBlackHeight string  `json:"evalBlackHeight"` // pre-formatted CSS height for the black overlay (e.g. "93.4%")
	EvalTextColor   string  `json:"evalTextColor"`   // hex color for the score label (light on a near-all-black bar, else dark)
	EvalReady       bool    `json:"evalReady"`       // false until first scored update
	EvalAnalyzing   bool    `json:"evalAnalyzing"`   // true while a job is running for this game
}

// Options controls per-job search budget.
type Options struct {
	Depth     int           // 0 → 18
	Movetime  time.Duration // 0 → 2s
	MaxJobAge time.Duration // safety ceiling on a single job; 0 → 30s
}

// Runner manages per-game and per-FEN analysis jobs.
//
//   - jobs is keyed by a caller-chosen cancellation slot ("jobKey"). The live
//     game uses gameID; history navigation uses "hist:"+gameID. Same key = the
//     new request cancels the prior.
//   - fenCache is keyed by FEN so history navigation can replay results
//     without re-analysing.
//   - gameLatest is keyed by gameID and tracks the most recent snapshot for
//     the live game so the move-broadcast path can copy it into outgoing
//     signals (preventing the eval bar from snapping back to 50/50 every
//     move while the new position is still being analysed).
type Runner struct {
	pool *uci.Pool
	opts Options

	mu         sync.Mutex
	nextID     uint64
	jobs       map[string]jobSlot
	fenCache   map[string]EvalSnapshot
	gameLatest map[string]EvalSnapshot
}

func NewRunner(pool *uci.Pool, opts Options) *Runner {
	if opts.Depth <= 0 {
		opts.Depth = 18
	}
	if opts.Movetime <= 0 {
		opts.Movetime = 2 * time.Second
	}
	if opts.MaxJobAge <= 0 {
		opts.MaxJobAge = 30 * time.Second
	}
	return &Runner{
		pool:       pool,
		opts:       opts,
		jobs:       make(map[string]jobSlot),
		fenCache:   make(map[string]EvalSnapshot),
		gameLatest: make(map[string]EvalSnapshot),
	}
}

// Start cancels any in-flight job for jobKey and launches a fresh analysis of
// fen. gameID, when non-empty, tags the snapshot for game-keyed lookup (used
// by the live-move broadcast path). onUpdate is called for each progress
// update including the final one where EvalAnalyzing flips false. Safe to
// call from any goroutine.
func (r *Runner) Start(jobKey, gameID, fen string, whiteToMove bool, onUpdate func(EvalSnapshot)) {
	r.mu.Lock()
	if prev, ok := r.jobs[jobKey]; ok {
		prev.cancel()
	}
	jobCtx, cancel := context.WithTimeout(context.Background(), r.opts.MaxJobAge)
	r.nextID++
	myID := r.nextID
	done := make(chan struct{})
	r.jobs[jobKey] = jobSlot{id: myID, cancel: cancel, done: done}
	r.mu.Unlock()

	go r.run(jobCtx, cancel, done, myID, jobKey, gameID, fen, whiteToMove, onUpdate)
}

func (r *Runner) run(ctx context.Context, myCancel context.CancelFunc, done chan struct{}, myID uint64, jobKey, gameID, fen string, whiteToMove bool, onUpdate func(EvalSnapshot)) {
	defer close(done)
	defer func() {
		// Only clear the slot — and only ship the terminal "analyzing=false"
		// frame — if the slot still belongs to us. Two ways we may no longer
		// own it: a successor Start replaced the slot, or an explicit
		// CancelHistory deleted it. In both cases a fresh state has already
		// been published (new job started, or a return-to-live direct push),
		// and broadcasting the cancelled job's last fenCache snapshot would
		// race that newer state on every subscriber's client (e.g. snap the
		// eval bar back to the historical position right after the user
		// returns to live).
		r.mu.Lock()
		stillOurs := false
		if cur, ok := r.jobs[jobKey]; ok && cur.id == myID {
			stillOurs = true
			delete(r.jobs, jobKey)
		}
		var snap EvalSnapshot
		var have bool
		if stillOurs {
			snap, have = r.fenCache[fen]
		}
		r.mu.Unlock()
		// Always release this run's own ctx so its timer fires no later than now.
		myCancel()
		if stillOurs && have && snap.EvalAnalyzing {
			snap.EvalAnalyzing = false
			r.cache(gameID, fen, snap)
			onUpdate(snap)
		}
	}()

	// Announce that a job started so the UI can show a spinner even before
	// the first depth iteration completes. Seed visible fields from the
	// FEN cache (re-analysing a known position), then the game's latest
	// snapshot (the previous position's eval kept as a continuity
	// placeholder), then neutral defaults. Without a seed the snapshot
	// would ship empty-string EvalBlackHeight/EvalDisplay and the bar
	// would render as full-white nothingness.
	startSnap := r.seedSnapshot(gameID, fen)
	startSnap.EvalAnalyzing = true
	r.cache(gameID, fen, startSnap)
	onUpdate(startSnap)

	eng, err := r.pool.Acquire(ctx)
	if err != nil {
		return
	}
	var releaseErr error
	defer func() { r.pool.Release(eng, releaseErr) }()

	movetimeMs := int(r.opts.Movetime / time.Millisecond)
	ch, err := eng.Analyze(ctx, fen, uci.AnalyzeOptions{
		MultiPV:   1,
		Depth:     r.opts.Depth,
		MovetimeM: movetimeMs,
	})
	if err != nil {
		releaseErr = err
		return
	}

	for u := range ch {
		if u.MultiPV > 1 {
			continue
		}
		if u.Final {
			break
		}
		snap := toSnapshot(u, whiteToMove)
		snap.EvalReady = true
		snap.EvalAnalyzing = true
		r.cache(gameID, fen, snap)
		onUpdate(snap)
	}
}

// LatestForFEN returns the cached snapshot for fen, used by history navigation.
func (r *Runner) LatestForFEN(fen string) (EvalSnapshot, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.fenCache[fen]
	return s, ok
}

// LatestForGame returns the most recent snapshot emitted for gameID (across
// whatever live position was analysed most recently). Used by the move
// broadcast path to keep the eval bar populated while the new position is
// still being analysed.
func (r *Runner) LatestForGame(gameID string) (EvalSnapshot, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.gameLatest[gameID]
	return s, ok
}

// Forget drops per-game cached state and cancels any in-flight jobs for
// gameID — both the live slot and the history-navigation slot. FEN cache
// entries are kept; they remain valid for replay during history navigation.
func (r *Runner) Forget(gameID string) {
	r.mu.Lock()
	if slot, ok := r.jobs[gameID]; ok {
		slot.cancel()
		delete(r.jobs, gameID)
	}
	hk := histKey(gameID)
	if slot, ok := r.jobs[hk]; ok {
		slot.cancel()
		delete(r.jobs, hk)
	}
	delete(r.gameLatest, gameID)
	r.mu.Unlock()
}

// RecordTerminal cancels any in-flight live job for gameID and records snap
// as the authoritative final eval for the game. Used when a move ends the
// game — there's nothing left for the engine to search and the outcome is
// already known from game state. Cancelling first prevents a still-running
// pre-terminal job from later broadcasting a stale eval that would overwrite
// the terminal snapshot on subscribers.
func (r *Runner) RecordTerminal(gameID, fen string, snap EvalSnapshot) {
	r.mu.Lock()
	if slot, ok := r.jobs[gameID]; ok {
		slot.cancel()
		delete(r.jobs, gameID)
	}
	r.mu.Unlock()
	r.cache(gameID, fen, snap)
}

// CancelHistory cancels any in-flight history-navigation analysis for gameID
// and waits for its goroutine to fully drain (defer included) before
// returning. The wait matters: a straggler update emitted between the engine
// loop and the cancel signal would otherwise race past the caller's follow-up
// state push (e.g. the return-to-live eval) and overwrite it on every
// subscriber's client.
func (r *Runner) CancelHistory(gameID string) {
	r.mu.Lock()
	hk := histKey(gameID)
	slot, ok := r.jobs[hk]
	if ok {
		slot.cancel()
		delete(r.jobs, hk)
	}
	r.mu.Unlock()
	if ok && slot.done != nil {
		<-slot.done
	}
}

func (r *Runner) cache(gameID, fen string, snap EvalSnapshot) {
	r.mu.Lock()
	r.fenCache[fen] = snap
	if gameID != "" {
		r.gameLatest[gameID] = snap
	}
	r.mu.Unlock()
}

func (r *Runner) seedSnapshot(gameID, fen string) EvalSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.fenCache[fen]; ok {
		return s
	}
	if gameID != "" {
		if s, ok := r.gameLatest[gameID]; ok {
			return s
		}
	}
	return EvalSnapshot{
		EvalWp:          50,
		EvalDisplay:     "0.00",
		EvalBlackHeight: "50%",
		EvalTextColor:   "#404040",
	}
}

func toSnapshot(u uci.Update, whiteToMove bool) EvalSnapshot {
	snap := EvalSnapshot{EvalDepth: u.Depth}
	if u.Mate != 0 {
		mate := u.Mate
		if !whiteToMove {
			mate = -mate
		}
		// UCI `score mate N` reports N in full moves (per spec), so no
		// ply→move conversion here. Sign is from side-to-move POV before the
		// whiteToMove flip above.
		snap.EvalMate = mate
		if mate > 0 {
			snap.EvalWp = 100
			snap.EvalDisplay = fmt.Sprintf("M%d", mate)
		} else {
			snap.EvalWp = 0
			snap.EvalDisplay = fmt.Sprintf("-M%d", -mate)
		}
		snap.EvalBlackHeight = fmt.Sprintf("%.1f%%", 100-snap.EvalWp)
		snap.EvalTextColor = textColorForWp(snap.EvalWp)
		return snap
	}
	cp := u.ScoreCP
	if !whiteToMove {
		cp = -cp
	}
	snap.EvalCp = cp
	snap.EvalWp = winPercent(cp)
	snap.EvalDisplay = fmt.Sprintf("%+.2f", float64(cp)/100)
	snap.EvalBlackHeight = fmt.Sprintf("%.1f%%", 100-snap.EvalWp)
	snap.EvalTextColor = textColorForWp(snap.EvalWp)
	return snap
}

// textColorForWp picks the score-label color so it remains readable as the
// bar shifts. The label sits at the very bottom of the bar; that area is
// white whenever any white share remains, except in the extreme case where
// black covers the entire bar (mate-against-white). Threshold is conservative
// so the colour flip only happens when the white sliver is too small to
// reliably contain the text.
func textColorForWp(wp float64) string {
	if wp < 10 {
		return "#f5f5f5"
	}
	return "#404040"
}

// winPercent converts centipawn eval (white POV) to win probability in [0,100]
// using the Lichess formula.
func winPercent(cp int) float64 {
	wp := 50 + 50*(2/(1+math.Exp(-0.00368208*float64(cp)))-1)
	if wp < 0 {
		return 0
	}
	if wp > 100 {
		return 100
	}
	return wp
}
