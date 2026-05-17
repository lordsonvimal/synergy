package analysis

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/ctxkeys"
)

// Middleware attaches the Runner (may be nil) to the request context. A nil
// runner is valid — handlers should treat it as "analysis disabled" and skip
// silently rather than erroring.
func Middleware(r *Runner) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ctxkeys.AnalysisRunnerKey, r)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// FromContext returns the Runner attached by Middleware. ok=false means
// analysis is not configured on this server; callers should no-op.
func FromContext(ctx context.Context) (*Runner, bool) {
	r, ok := ctx.Value(ctxkeys.AnalysisRunnerKey).(*Runner)
	if !ok || r == nil {
		return nil, false
	}
	return r, true
}
