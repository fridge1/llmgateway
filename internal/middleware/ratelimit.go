package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/metrics"
	"golang.org/x/time/rate"
)

// RateLimiter is a per-key token-bucket rate limiter with automatic cleanup of idle entries.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*entry
	rate     rate.Limit
	burst    int
	stopCh   chan struct{}
}

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a RateLimiter. Set rps <= 0 to disable rate limiting.
// cleanupInterval controls how often idle entries are removed (default 1 min).
func NewRateLimiter(rps float64, burst int, cleanupInterval time.Duration) *RateLimiter {
	if burst <= 0 {
		burst = int(rps) * 2
		if burst < 1 {
			burst = 1
		}
	}
	if cleanupInterval <= 0 {
		cleanupInterval = 1 * time.Minute
	}
	rl := &RateLimiter{
		limiters: make(map[string]*entry),
		rate:     rate.Limit(rps),
		burst:    burst,
		stopCh:   make(chan struct{}),
	}
	go rl.cleanup(cleanupInterval)
	return rl
}

// Allow checks whether a request identified by key is allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	e, ok := rl.limiters[key]
	if !ok {
		e = &entry{
			limiter: rate.NewLimiter(rl.rate, rl.burst),
		}
		rl.limiters[key] = e
	}
	e.lastSeen = time.Now()
	rl.mu.Unlock()
	return e.limiter.Allow()
}

// Stop terminates the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

func (rl *RateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-3 * time.Minute)
			for k, e := range rl.limiters {
				if e.lastSeen.Before(cutoff) {
					delete(rl.limiters, k)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

// CompositeRateLimitConfig holds configuration for multi-dimension rate limiting.
type CompositeRateLimitConfig struct {
	PerIPRPS    float64
	PerIPBurst  int
	PerUserRPS  float64
	PerUserBurst int
	PerKeyRPS   float64
	PerKeyBurst int
	CleanupInterval time.Duration
}

// CompositeRateLimiter chains multiple per-dimension rate limiters.
type CompositeRateLimiter struct {
	ipLimiter   *RateLimiter
	userLimiter *RateLimiter
	keyLimiter  *RateLimiter
}

// Stop terminates all background cleanup goroutines.
func (c *CompositeRateLimiter) Stop() {
	if c.ipLimiter != nil {
		c.ipLimiter.Stop()
	}
	if c.userLimiter != nil {
		c.userLimiter.Stop()
	}
	if c.keyLimiter != nil {
		c.keyLimiter.Stop()
	}
}

// RateLimit returns an HTTP middleware that limits requests by client IP,
// and optionally by user ID and API key ID.
// If all rps values are <= 0, the middleware is a no-op passthrough.
func RateLimit(cfg CompositeRateLimitConfig) (func(http.Handler) http.Handler, *CompositeRateLimiter) {
	if cfg.PerIPRPS <= 0 && cfg.PerUserRPS <= 0 && cfg.PerKeyRPS <= 0 {
		return func(next http.Handler) http.Handler { return next }, nil
	}

	comp := &CompositeRateLimiter{}
	if cfg.PerIPRPS > 0 {
		comp.ipLimiter = NewRateLimiter(cfg.PerIPRPS, cfg.PerIPBurst, cfg.CleanupInterval)
	}
	if cfg.PerUserRPS > 0 {
		comp.userLimiter = NewRateLimiter(cfg.PerUserRPS, cfg.PerUserBurst, cfg.CleanupInterval)
	}
	if cfg.PerKeyRPS > 0 {
		comp.keyLimiter = NewRateLimiter(cfg.PerKeyRPS, cfg.PerKeyBurst, cfg.CleanupInterval)
	}

	rateLimitedResponse := `{"error":{"message":"rate limit exceeded","type":"rate_limit_error","code":"rate_limited"}}`

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set rate limit headers.
			if cfg.PerIPRPS > 0 {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", cfg.PerIPRPS))
			}

			// Check IP rate limit.
			if comp.ipLimiter != nil {
				ip := httputil.ClientIP(r)
				if !comp.ipLimiter.Allow(ip) {
					metrics.IncrementRateLimitHitIP()
					w.Header().Set("Retry-After", "1")
					w.Header().Set("Content-Type", "application/json")
					http.Error(w, rateLimitedResponse, http.StatusTooManyRequests)
					return
				}
			}

			// Check user rate limit (if authenticated).
			if comp.userLimiter != nil {
				if userID, ok := r.Context().Value(admin.CtxUserIDKey).(string); ok && userID != "" {
					if !comp.userLimiter.Allow(userID) {
						metrics.IncrementRateLimitHitUser()
						w.Header().Set("Retry-After", "1")
						w.Header().Set("Content-Type", "application/json")
						http.Error(w, rateLimitedResponse, http.StatusTooManyRequests)
						return
					}
				}
			}

			// Check API key rate limit (if available in context).
			if comp.keyLimiter != nil {
				if keyID, ok := r.Context().Value(admin.CtxKeyIDKey).(string); ok && keyID != "" {
					if !comp.keyLimiter.Allow(keyID) {
						metrics.IncrementRateLimitHitAPIKey()
						w.Header().Set("Retry-After", "1")
						w.Header().Set("Content-Type", "application/json")
						http.Error(w, rateLimitedResponse, http.StatusTooManyRequests)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
	return mw, comp
}
