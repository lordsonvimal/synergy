package uci

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"sync"
)

// PoolConfig configures a Pool. Zero values pick sensible defaults.
type PoolConfig struct {
	BinaryPath string // empty → ResolveBinary
	Size       int    // 0 → min(NumCPU/2, 4)
	HashMB     int    // 0 → 64
	ThreadsPer int    // 0 → 1 (so pool size = parallelism)
}

// Pool owns a fixed set of Engines and hands them out one analysis at a time.
type Pool struct {
	cfg  PoolConfig
	idle chan *Engine

	mu     sync.Mutex
	closed bool
}

// NewPool spawns the configured number of engines and returns a ready Pool.
func NewPool(ctx context.Context, cfg PoolConfig) (*Pool, error) {
	if cfg.BinaryPath == "" {
		p, err := ResolveBinary()
		if err != nil {
			return nil, err
		}
		cfg.BinaryPath = p
	}
	if cfg.Size <= 0 {
		n := runtime.NumCPU() / 2
		if n < 1 {
			n = 1
		}
		if n > 4 {
			n = 4
		}
		cfg.Size = n
	}
	if cfg.HashMB <= 0 {
		cfg.HashMB = 64
	}
	if cfg.ThreadsPer <= 0 {
		cfg.ThreadsPer = 1
	}

	p := &Pool{
		cfg:  cfg,
		idle: make(chan *Engine, cfg.Size),
	}
	for i := 0; i < cfg.Size; i++ {
		eng, err := p.spawn(ctx)
		if err != nil {
			p.Close()
			return nil, err
		}
		p.idle <- eng
	}
	return p, nil
}

func (p *Pool) spawn(ctx context.Context) (*Engine, error) {
	eng, err := Spawn(ctx, p.cfg.BinaryPath)
	if err != nil {
		return nil, err
	}
	if err := eng.SetOption("Threads", strconv.Itoa(p.cfg.ThreadsPer)); err != nil {
		_ = eng.Close()
		return nil, err
	}
	if err := eng.SetOption("Hash", strconv.Itoa(p.cfg.HashMB)); err != nil {
		_ = eng.Close()
		return nil, err
	}
	if err := eng.IsReady(ctx); err != nil {
		_ = eng.Close()
		return nil, err
	}
	return eng, nil
}

// Acquire blocks until an engine is free or ctx is cancelled. Caller must call
// Release with the same engine when finished; on error, Release with err != nil
// to let the pool respawn it.
func (p *Pool) Acquire(ctx context.Context) (*Engine, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, errors.New("uci: pool is closed")
	}
	p.mu.Unlock()
	select {
	case eng := <-p.idle:
		return eng, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release returns the engine to the pool. If err != nil the engine is closed
// and a fresh one is spawned in its place (best-effort; on respawn failure the
// slot is dropped and pool capacity shrinks).
func (p *Pool) Release(eng *Engine, err error) {
	if eng == nil {
		return
	}
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed {
		_ = eng.Close()
		return
	}
	if err != nil {
		_ = eng.Close()
		newEng, spawnErr := p.spawn(context.Background())
		if spawnErr != nil {
			return
		}
		eng = newEng
	}
	p.idle <- eng
}

// Close shuts down all engines. Acquire calls after Close return an error.
func (p *Pool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()
	close(p.idle)
	for eng := range p.idle {
		_ = eng.Close()
	}
}

