package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/httputil"
)

// statusRecorder wraps http.ResponseWriter to capture the status code
// without buffering the body (important for SSE streams).
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher for SSE streaming support.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// AccessLog returns a middleware that logs each request with method, path,
// status code, latency, and request ID.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		ctx, identity := httputil.WithIdentitySlot(r.Context())
		next.ServeHTTP(rec, r.WithContext(ctx))

		// Skip logging for high-frequency image task status polling.
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/image/tasks") {
			return
		}

		slog.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.String("request_id", r.Header.Get("X-Request-ID")),
			slog.String("remote_addr", httputil.ClientIP(r)),
			slog.String("user", identity.Name),
			slog.String("user_kind", identity.Kind),
			slog.String("user_id", identity.UserID),
		)
	})
}
