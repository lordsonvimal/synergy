// Package uci speaks the Universal Chess Interface protocol to an external
// engine subprocess (Stockfish by default). One Engine owns one subprocess and
// serialises commands; for concurrency use Pool.
package uci

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Engine wraps a running UCI subprocess.
type Engine struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner

	name    string
	version string

	mu     sync.Mutex // serialises Analyze calls
	closed bool
}

// Spawn starts an engine subprocess at binaryPath and performs the UCI handshake.
func Spawn(ctx context.Context, binaryPath string) (*Engine, error) {
	cmd := exec.CommandContext(ctx, binaryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("uci: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("uci: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("uci: start %q: %w", binaryPath, err)
	}

	e := &Engine{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdout),
	}
	e.stdout.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	if err := e.handshake(ctx); err != nil {
		_ = e.Close()
		return nil, err
	}
	return e, nil
}

func (e *Engine) handshake(ctx context.Context) error {
	if err := e.send("uci"); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	for time.Now().Before(deadline) {
		if !e.stdout.Scan() {
			if err := e.stdout.Err(); err != nil {
				return fmt.Errorf("uci: read during handshake: %w", err)
			}
			return errors.New("uci: engine closed during handshake")
		}
		line := strings.TrimSpace(e.stdout.Text())
		switch {
		case strings.HasPrefix(line, "id name "):
			e.name = strings.TrimPrefix(line, "id name ")
		case strings.HasPrefix(line, "id author "):
			e.version = strings.TrimPrefix(line, "id author ")
		case line == "uciok":
			return e.IsReady(ctx)
		}
	}
	return errors.New("uci: handshake timed out")
}

// SetOption sends `setoption name <k> value <v>`. Caller must wait for IsReady.
func (e *Engine) SetOption(name, value string) error {
	return e.send(fmt.Sprintf("setoption name %s value %s", name, value))
}

// IsReady issues `isready` and blocks until `readyok`.
func (e *Engine) IsReady(ctx context.Context) error {
	if err := e.send("isready"); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !e.stdout.Scan() {
			if err := e.stdout.Err(); err != nil {
				return err
			}
			return errors.New("uci: engine closed waiting for readyok")
		}
		if strings.TrimSpace(e.stdout.Text()) == "readyok" {
			return nil
		}
	}
}

// Name returns the engine's reported "id name" string.
func (e *Engine) Name() string { return e.name }

func (e *Engine) send(line string) error {
	if _, err := io.WriteString(e.stdin, line+"\n"); err != nil {
		return fmt.Errorf("uci: write %q: %w", line, err)
	}
	return nil
}

// Close stops any in-flight search and shuts down the subprocess.
func (e *Engine) Close() error {
	if e.closed {
		return nil
	}
	e.closed = true
	_ = e.send("stop")
	_ = e.send("quit")
	_ = e.stdin.Close()
	done := make(chan error, 1)
	go func() { done <- e.cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		_ = e.cmd.Process.Kill()
		return <-done
	}
}
