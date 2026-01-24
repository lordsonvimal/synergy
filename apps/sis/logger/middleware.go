package logger

import (
	"regexp"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

var sensitiveRegex = regexp.MustCompile(`(?i)(access_token|auth_token|access_token_ttl|password|secret|api[_-]?key)=[^& \n]*`)

// Setup RequestID and Contextual logger, Fast Redaction and Structured Logging all in one pass.
func RedactedStructuredLogger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Setup Request ID and Contextual Logger
		rid := requestid.Get(c)
		l := logger.With().Str("request_id", rid).Logger()

		// Put logger into context so handlers can use logger.FromContext(ctx)
		c.Request = c.Request.WithContext(l.WithContext(c.Request.Context()))

		c.Next()

		// Post-Request logic (Logging)
		latency := time.Since(start)
		status := c.Writer.Status()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// Fast Redaction
		displayPath := path
		if rawQuery != "" {
			redactedQuery := sensitiveRegex.ReplaceAllString(rawQuery, "$1=[REDACTED]")
			displayPath = path + "?" + redactedQuery
		}

		ctx := c.Request.Context()
		// Determine Log Level based on Status Code
		var event *zerolog.Event
		switch {
		case status >= 500:
			event = Error(ctx)
		case status >= 400:
			event = Warn(ctx)
		default:
			event = Info(ctx)
		}

		event.
			Int("status", status).
			Str("method", c.Request.Method).
			Str("path", displayPath).
			Str("ip", c.ClientIP()).
			Dur("latency", latency).
			Str("user_agent", c.Request.UserAgent()).
			Msg("HTTP Request")
	}
}
