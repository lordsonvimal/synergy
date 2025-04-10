package logger

import (
	"context"
	"fmt"
	"strings"
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

		fullHandlerName := c.HandlerName()

		// Extract only the function name
		handlerParts := strings.Split(fullHandlerName, ".")
		handlerName := handlerParts[len(handlerParts)-1]

		// Extract trace and span IDs
		spanCtx := span.SpanContext()
		fields := map[string]any{
			"request_id": requestID,
			"ip":         c.ClientIP(),
			// "user_agent": c.Request.UserAgent(),
			"trace_id": spanCtx.TraceID().String(),
			"span_id":  spanCtx.SpanID().String(),
		}

		req := fmt.Sprintf("method=%s Handler=%s Path=%s Params=%s", c.Request.Method, handlerName, c.Request.URL.Path, c.Params)

		// Create logger instance with request context
		log := GetLogger().WithContext(c.Request.Context())
		ctx := context.WithValue(c.Request.Context(), LoggerKey, log)
		c.Request = c.Request.WithContext(ctx)

		log.Info(c.Request.Context(), req, fields)

		// Add logger and request ID to context
		c.Set(string(LoggerKey), log)
		c.Set("request_id", requestID)

		c.Next()

		// Log response with duration
		status := c.Writer.Status()
		duration := time.Since(start)
		route := c.FullPath()

		result := fmt.Sprintf("method=%s Handler=%s status=%d duration=%s route=%s", c.Request.Method, handlerName, status, duration.String(), route)

		if status >= 200 && status < 400 {
			log.Info(c.Request.Context(), result, nil)
		}

		if status >= 400 {
			log.Error(c.Request.Context(), result, nil)
		}
	}
}
