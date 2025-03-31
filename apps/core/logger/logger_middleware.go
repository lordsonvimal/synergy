package logger

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// LoggerMiddleware injects logger and ensures trace_id generation
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate or use existing request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Check if trace exists, else create a new one
		span := trace.SpanFromContext(c.Request.Context())
		if !span.SpanContext().IsValid() {
			// Generate a new trace if missing
			tracer := otel.Tracer("synergy-tracer")
			ctx, newSpan := tracer.Start(c.Request.Context(), "http-request")
			defer newSpan.End()

			// Update the context with the new trace
			c.Request = c.Request.WithContext(ctx)
			span = newSpan
		}

		// Extract trace and span IDs
		// spanCtx := span.SpanContext()
		fields := map[string]any{
			// "request_id": requestID,
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			// "ip":     c.ClientIP(),
			"params": c.Params,
			// "user_agent": c.Request.UserAgent(),
			// "trace_id":   spanCtx.TraceID().String(),
			// "span_id":    spanCtx.SpanID().String(),
		}

		// Create logger instance with request context
		log := GetLogger().WithContext(c.Request.Context())
		ctx := context.WithValue(c.Request.Context(), LoggerKey, log)
		c.Request = c.Request.WithContext(ctx)

		log.Info(c.Request.Context(), "Incoming request", fields)

		// Add logger and request ID to context
		c.Set(string(LoggerKey), log)
		c.Set("request_id", requestID)

		c.Next()

		// Log response with duration
		status := c.Writer.Status()
		duration := time.Since(start)
		route := c.FullPath()

		log.Info(c.Request.Context(), "Request completed", map[string]any{
			"duration": duration.String(),
			"method":   c.Request.Method,
			"status":   status,
			"route":    route,
		})
	}
}
