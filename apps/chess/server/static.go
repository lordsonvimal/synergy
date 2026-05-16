package server

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ServeStatic returns a handler that serves files from distDir under a route
// like /static/*filepath. It does two things on top of a plain file server:
//
//  1. If the client accepts gzip and a precompressed sibling (<path>.gz) exists
//     on disk, that smaller file is served with Content-Encoding: gzip.
//  2. For any served file (compressed or not) it sets an aggressive
//     immutable Cache-Control header when the filename looks content-hashed
//     (e.g. style.D2C357B3.css). Unhashed names (style.css, board.js — only
//     produced by dev builds) get no-cache so developers see edits without
//     hard-reloads.
//
// The build manifest itself is intentionally not served from here — it lives
// inside distDir but only Go reads it. Keeping it private avoids leaking the
// mapping and avoids the cache-busting paradox of a "stable name with mutable
// bytes".
func ServeStatic(distDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rel := strings.TrimPrefix(c.Param("filepath"), "/")
		if rel == "" || rel == "manifest.json" {
			c.Status(http.StatusNotFound)
			return
		}
		clean := filepath.Clean(rel)
		if strings.HasPrefix(clean, "..") {
			c.Status(http.StatusNotFound)
			return
		}
		full := filepath.Join(distDir, clean)

		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			c.Status(http.StatusNotFound)
			return
		}

		// Resolve Content-Type from the *original* filename — the gzipped
		// sibling has a .gz extension which mime.TypeByExtension would
		// (correctly) report as application/gzip and break browsers.
		ctype := mime.TypeByExtension(filepath.Ext(full))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		c.Header("Content-Type", ctype)
		if hasContentHash(filepath.Base(clean)) {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "no-cache")
		}
		c.Header("Vary", "Accept-Encoding")

		if acceptsGzip(c.Request.Header.Get("Accept-Encoding")) {
			gz := full + ".gz"
			if gzInfo, err := os.Stat(gz); err == nil && !gzInfo.IsDir() {
				c.Header("Content-Encoding", "gzip")
				c.File(gz)
				return
			}
		}
		c.File(full)
	}
}

// hasContentHash returns true when a basename looks like "<stem>.<HASH>.<ext>"
// where HASH is 8 hex digits — the convention build-assets.mjs uses for
// production-mode fingerprinted filenames.
func hasContentHash(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) < 3 {
		return false
	}
	candidate := parts[len(parts)-2]
	if len(candidate) != 8 {
		return false
	}
	for i := 0; i < len(candidate); i++ {
		c := candidate[i]
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func acceptsGzip(header string) bool {
	for _, enc := range strings.Split(header, ",") {
		token := strings.TrimSpace(enc)
		if token == "gzip" || strings.HasPrefix(token, "gzip;") {
			return true
		}
	}
	return false
}
