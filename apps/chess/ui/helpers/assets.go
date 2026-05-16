package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// manifest maps logical asset names (e.g. "style.css", "js/board.js") to
// content-hashed filenames produced by the build pipeline
// (e.g. "style.D2C357B3.css", "js/board.S3BBSFNV.js"). It is re-read whenever
// the file's mtime changes on disk so a rebuild doesn't strand a running
// server with stale hashed filenames (the symptom is /static/*.js 404s on
// page loads after a `yarn build` that ran while the server was up).
var (
	manifestMu      sync.RWMutex
	manifestPath    string
	manifestData    map[string]string
	manifestModTime time.Time
)

// LoadAssetManifest reads dist/manifest.json into memory and remembers the
// path so subsequent Asset() lookups can refresh on mtime change.
func LoadAssetManifest(distDir string) error {
	path := filepath.Join(distDir, "manifest.json")
	manifestMu.Lock()
	manifestPath = path
	manifestMu.Unlock()
	return refreshManifest()
}

func refreshManifest() error {
	manifestMu.RLock()
	path := manifestPath
	prev := manifestModTime
	manifestMu.RUnlock()
	if path == "" {
		return fmt.Errorf("asset manifest path not set; call LoadAssetManifest at startup")
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("asset manifest %s: %w", path, err)
	}
	if !info.ModTime().After(prev) {
		return nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("asset manifest %s: %w", path, err)
	}
	m := map[string]string{}
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("asset manifest %s: %w", path, err)
	}

	manifestMu.Lock()
	manifestData = m
	manifestModTime = info.ModTime()
	manifestMu.Unlock()
	return nil
}

// Asset returns the served path for a logical asset name, looking up the
// build manifest (re-reading if the file changed since last lookup). Missing
// entries fall back to the logical name so dev builds (CHESS_BUILD_DEV=1 —
// no hashing) and any not-yet-built asset still resolve.
func Asset(logical string) string {
	_ = refreshManifest()
	manifestMu.RLock()
	hashed, ok := manifestData[logical]
	manifestMu.RUnlock()
	if ok {
		return "/static/" + hashed
	}
	return "/static/" + logical
}
