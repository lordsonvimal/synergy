package uci

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const EnvBinaryPath = "CHESSLEAP_STOCKFISH_PATH"

var ErrBinaryNotFound = errors.New("uci: stockfish binary not found")

// ResolveBinary locates a Stockfish executable.
// Order: env override → sibling engines/ dir next to the running binary → $PATH.
func ResolveBinary() (string, error) {
	if p := os.Getenv(EnvBinaryPath); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		name := "stockfish-" + runtime.GOOS + "-" + runtime.GOARCH
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		candidate := filepath.Join(dir, "engines", name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		plain := filepath.Join(dir, "engines", "stockfish")
		if runtime.GOOS == "windows" {
			plain += ".exe"
		}
		if _, err := os.Stat(plain); err == nil {
			return plain, nil
		}
	}

	if p, err := exec.LookPath("stockfish"); err == nil {
		return p, nil
	}

	return "", ErrBinaryNotFound
}
