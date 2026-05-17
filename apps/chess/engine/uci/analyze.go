package uci

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// AnalyzeOptions controls one Analyze call. Zero values mean "engine default".
type AnalyzeOptions struct {
	MultiPV   int    // number of PV lines to request (1..N)
	Depth     int    // max search depth; 0 = no limit
	MovetimeM int    // wall-clock budget in milliseconds; 0 = no limit
	Nodes     int    // node budget; 0 = no limit
	Moves     string // optional space-separated UCI moves appended to the position (e.g. for hypothetical lines)
}

// Update is one snapshot of engine thinking. Many arrive per Analyze call as
// depth increases; the final Update has Final=true and BestMove populated.
type Update struct {
	MultiPV  int      // 1-indexed PV rank this update describes (0 = not a PV info line)
	Depth    int      // search depth in plies
	ScoreCP  int      // centipawns from side-to-move POV; ignored when Mate != 0
	Mate     int      // signed mate distance in plies; 0 means no forced mate found
	Nodes    int64    // nodes searched
	NPS      int64    // nodes per second
	TimeMS   int      // ms elapsed
	PV       []string // principal variation as UCI moves
	BestMove string   // populated only on the terminal Update
	Final    bool
}

// Analyze streams Updates for the position `fen`. The channel closes when the
// engine emits `bestmove` or the context is cancelled. Concurrent calls on the
// same Engine are serialised.
func (e *Engine) Analyze(ctx context.Context, fen string, opts AnalyzeOptions) (<-chan Update, error) {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil, errors.New("uci: engine is closed")
	}

	if opts.MultiPV > 0 {
		if err := e.SetOption("MultiPV", strconv.Itoa(opts.MultiPV)); err != nil {
			e.mu.Unlock()
			return nil, err
		}
	}
	if err := e.IsReady(ctx); err != nil {
		e.mu.Unlock()
		return nil, err
	}

	posCmd := "position fen " + fen
	if opts.Moves != "" {
		posCmd += " moves " + opts.Moves
	}
	if err := e.send(posCmd); err != nil {
		e.mu.Unlock()
		return nil, err
	}

	var goParts []string
	goParts = append(goParts, "go")
	if opts.Depth > 0 {
		goParts = append(goParts, "depth", strconv.Itoa(opts.Depth))
	}
	if opts.MovetimeM > 0 {
		goParts = append(goParts, "movetime", strconv.Itoa(opts.MovetimeM))
	}
	if opts.Nodes > 0 {
		goParts = append(goParts, "nodes", strconv.FormatInt(int64(opts.Nodes), 10))
	}
	if len(goParts) == 1 {
		goParts = append(goParts, "infinite")
	}
	if err := e.send(strings.Join(goParts, " ")); err != nil {
		e.mu.Unlock()
		return nil, err
	}

	out := make(chan Update, 8)
	go func() {
		defer close(out)
		defer e.mu.Unlock()

		stopOnCancel := make(chan struct{})
		defer close(stopOnCancel)
		go func() {
			select {
			case <-ctx.Done():
				_ = e.send("stop")
			case <-stopOnCancel:
			}
		}()

		for e.stdout.Scan() {
			line := strings.TrimSpace(e.stdout.Text())
			if line == "" {
				continue
			}
			switch {
			case strings.HasPrefix(line, "info "):
				if u, ok := parseInfo(line); ok {
					select {
					case out <- u:
					case <-ctx.Done():
						return
					}
				}
			case strings.HasPrefix(line, "bestmove "):
				final := Update{Final: true}
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					final.BestMove = fields[1]
				}
				select {
				case out <- final:
				case <-ctx.Done():
				}
				return
			}
		}
	}()

	return out, nil
}

// parseInfo parses a UCI "info" line. Returns ok=false for info strings that
// carry no scored PV (e.g. "info string ...").
func parseInfo(line string) (Update, bool) {
	fields := strings.Fields(line)
	u := Update{}
	hasScore := false
	i := 1 // skip "info"
	for i < len(fields) {
		tok := fields[i]
		switch tok {
		case "depth":
			i++
			if i < len(fields) {
				u.Depth, _ = strconv.Atoi(fields[i])
			}
		case "multipv":
			i++
			if i < len(fields) {
				u.MultiPV, _ = strconv.Atoi(fields[i])
			}
		case "score":
			i++
			if i+1 < len(fields) {
				kind := fields[i]
				val, _ := strconv.Atoi(fields[i+1])
				switch kind {
				case "cp":
					u.ScoreCP = val
					hasScore = true
				case "mate":
					u.Mate = val
					hasScore = true
				}
				i++
			}
		case "nodes":
			i++
			if i < len(fields) {
				u.Nodes, _ = strconv.ParseInt(fields[i], 10, 64)
			}
		case "nps":
			i++
			if i < len(fields) {
				u.NPS, _ = strconv.ParseInt(fields[i], 10, 64)
			}
		case "time":
			i++
			if i < len(fields) {
				u.TimeMS, _ = strconv.Atoi(fields[i])
			}
		case "pv":
			u.PV = append(u.PV, fields[i+1:]...)
			i = len(fields)
		case "string":
			// "info string ..." has no machine-readable score.
			return Update{}, false
		}
		i++
	}
	if !hasScore && len(u.PV) == 0 {
		return Update{}, false
	}
	if u.MultiPV == 0 {
		u.MultiPV = 1
	}
	return u, true
}

// FormatScore renders a score in the conventional "+0.42" / "M5" form,
// from white's POV given side-to-move.
func FormatScore(u Update, whiteToMove bool) string {
	if u.Mate != 0 {
		sign := u.Mate
		if !whiteToMove {
			sign = -sign
		}
		if sign > 0 {
			return fmt.Sprintf("M%d", (u.Mate+1)/2)
		}
		return fmt.Sprintf("-M%d", (-u.Mate+1)/2)
	}
	cp := u.ScoreCP
	if !whiteToMove {
		cp = -cp
	}
	return fmt.Sprintf("%+.2f", float64(cp)/100)
}
