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
	"reflect"
	"sync"
	"time"

	"github.com/lordsonvimal/synergy/apps/chess/engine/uci"
)

// sameFunc reports whether two CancelFuncs are the same value. context.CancelFunc
// is not directly comparable so we compare via reflect.Value pointers.
func sameFunc(a, b context.CancelFunc) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
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
	jobs       map[string]context.CancelFunc
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
		jobs:       make(map[string]context.CancelFunc),
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
	if cancel, ok := r.jobs[jobKey]; ok {
		cancel()
	}
	jobCtx, cancel := context.WithTimeout(context.Background(), r.opts.MaxJobAge)
	r.jobs[jobKey] = cancel
	r.mu.Unlock()

	go r.run(jobCtx, cancel, jobKey, gameID, fen, whiteToMove, onUpdate)
}

func (r *Runner) run(ctx context.Context, myCancel context.CancelFunc, jobKey, gameID, fen string, whiteToMove bool, onUpdate func(EvalSnapshot)) {
	defer func() {
		// Only clear the slot if it still belongs to us. A faster successor
		// Start() may have already replaced it; clobbering would cancel the
		// new job (it shares this jobKey) and leave the eval bar stuck until
		// the *next* move triggered another Start.
		r.mu.Lock()
		if cur, ok := r.jobs[jobKey]; ok && cur != nil && sameFunc(cur, myCancel) {
			delete(r.jobs, jobKey)
		}
		snap, ok := r.fenCache[fen]
		r.mu.Unlock()
		// Always release this run's own ctx so its timer fires no later than now.
		myCancel()
		if ok && snap.EvalAnalyzing {
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

// Forget drops per-game cached state and cancels the live job for gameID.
// FEN cache entries are kept — they remain valid for history navigation.
func (r *Runner) Forget(gameID string) {
	r.mu.Lock()
	if cancel, ok := r.jobs[gameID]; ok {
		cancel()
		delete(r.jobs, gameID)
	}
	delete(r.gameLatest, gameID)
	r.mu.Unlock()
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
