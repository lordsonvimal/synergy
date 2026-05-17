package uci

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

const startposFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

func TestParseInfo(t *testing.T) {
	cases := []struct {
		name string
		line string
		want Update
		ok   bool
	}{
		{
			name: "centipawn pv",
			line: "info depth 12 seldepth 18 multipv 1 score cp 42 nodes 12345 nps 100000 time 123 pv e2e4 e7e5 g1f3",
			want: Update{MultiPV: 1, Depth: 12, ScoreCP: 42, Nodes: 12345, NPS: 100000, TimeMS: 123, PV: []string{"e2e4", "e7e5", "g1f3"}},
			ok:   true,
		},
		{
			name: "mate",
			line: "info depth 20 multipv 2 score mate 3 pv d1h5 g7g6 h5e5",
			want: Update{MultiPV: 2, Depth: 20, Mate: 3, PV: []string{"d1h5", "g7g6", "h5e5"}},
			ok:   true,
		},
		{
			name: "info string is skipped",
			line: "info string NNUE evaluation using nn-x.nnue",
			ok:   false,
		},
		{
			name: "multipv defaults to 1",
			line: "info depth 5 score cp 0 pv e2e4",
			want: Update{MultiPV: 1, Depth: 5, ScoreCP: 0, PV: []string{"e2e4"}},
			ok:   true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseInfo(tc.line)
			if ok != tc.ok {
				t.Fatalf("ok=%v want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
}

// TestSpawnAnalyzeMultiPV is an integration test against a real Stockfish.
// Skipped when no binary is resolvable.
func TestSpawnAnalyzeMultiPV(t *testing.T) {
	bin, err := ResolveBinary()
	if errors.Is(err, ErrBinaryNotFound) {
		t.Skip("stockfish binary not available; skipping integration test")
	}
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eng, err := Spawn(ctx, bin)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	defer eng.Close()

	if err := eng.SetOption("Threads", "1"); err != nil {
		t.Fatalf("setoption Threads: %v", err)
	}
	if err := eng.IsReady(ctx); err != nil {
		t.Fatalf("isready: %v", err)
	}

	ch, err := eng.Analyze(ctx, startposFEN, AnalyzeOptions{MultiPV: 5, Depth: 10})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}

	seen := map[int]bool{}
	var final Update
	for u := range ch {
		if u.Final {
			final = u
			continue
		}
		if u.MultiPV >= 1 && u.MultiPV <= 5 {
			seen[u.MultiPV] = true
		}
	}
	if !final.Final || final.BestMove == "" {
		t.Fatalf("no bestmove in final update: %+v", final)
	}
	for pv := 1; pv <= 5; pv++ {
		if !seen[pv] {
			t.Errorf("missing PV rank %d (saw %v)", pv, seen)
		}
	}
}

func TestPoolAcquireRelease(t *testing.T) {
	bin, err := ResolveBinary()
	if errors.Is(err, ErrBinaryNotFound) {
		t.Skip("stockfish binary not available; skipping integration test")
	}
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, PoolConfig{BinaryPath: bin, Size: 2})
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()

	eng, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if eng == nil {
		t.Fatal("acquire returned nil engine")
	}
	pool.Release(eng, nil)
}
