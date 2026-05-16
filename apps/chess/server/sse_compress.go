package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

// disableSSETimeouts clears the per-request read/write deadlines for the
// current connection. The base http.Server has a 1-minute WriteTimeout to
// protect normal endpoints from slow clients; without clearing it on SSE
// handlers, every long-lived stream is killed at the 60-second mark.
func disableSSETimeouts(c *gin.Context) {
	rc := http.NewResponseController(c.Writer)
	_ = rc.SetWriteDeadline(time.Time{})
	_ = rc.SetReadDeadline(time.Time{})
}

// brotliSSEWriter wraps gin.ResponseWriter so all Write/WriteString/Flush calls
// flow through a brotli encoder. Flush() emits a brotli sync flush so the
// browser receives bytes at every SSE frame boundary.
type brotliSSEWriter struct {
	gin.ResponseWriter
	bw *brotli.Writer
}

func (w *brotliSSEWriter) Write(p []byte) (int, error) {
	return w.bw.Write(p)
}

func (w *brotliSSEWriter) WriteString(s string) (int, error) {
	return w.bw.Write([]byte(s))
}

func (w *brotliSSEWriter) Flush() {
	_ = w.bw.Flush()
	w.ResponseWriter.Flush()
}

func (w *brotliSSEWriter) close() {
	_ = w.bw.Close()
	w.ResponseWriter.Flush()
}

// maybeBrotliSSE wraps c.Writer with a brotli encoder if the client advertises
// "br" in Accept-Encoding. Must be called after other response headers are set
// but before the first Write/Flush. Returns a cleanup func that flushes and
// closes the encoder — defer it for the lifetime of the SSE handler.
//
// No-op (returns a no-op cleanup) when the client doesn't accept brotli, so
// callers can use it unconditionally.
func maybeBrotliSSE(c *gin.Context) func() {
	if !acceptsBrotli(c.Request.Header.Get("Accept-Encoding")) {
		return func() {}
	}
	c.Header("Content-Encoding", "br")
	// Vary so caches/proxies don't serve a brotli body to a non-brotli client.
	c.Header("Vary", "Accept-Encoding")

	// BestSpeed (level 1): SSE flushes frequently and the wins come from
	// repetition across frames, not from squeezing each frame hard. Higher
	// levels burn CPU per flush for marginal gains without a shared dictionary.
	bw := brotli.NewWriterLevel(c.Writer, brotli.BestSpeed)
	wrapped := &brotliSSEWriter{ResponseWriter: c.Writer, bw: bw}
	c.Writer = wrapped
	return wrapped.close
}

// acceptsBrotli reports whether the Accept-Encoding header contains the "br"
// coding. Tolerates q-values and whitespace; ignores q=0 (explicit reject).
func acceptsBrotli(h string) bool {
	for _, part := range strings.Split(h, ",") {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		name := token
		params := ""
		if i := strings.Index(token, ";"); i >= 0 {
			name = strings.TrimSpace(token[:i])
			params = token[i+1:]
		}
		if !strings.EqualFold(name, "br") {
			continue
		}
		// Reject if q=0.
		for _, p := range strings.Split(params, ";") {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "q=") && strings.TrimSpace(p[2:]) == "0" {
				return false
			}
		}
		return true
	}
	return false
}
