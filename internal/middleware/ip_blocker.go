package middleware

import (
	"log"
	"net/http"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/metrics"
)

// IPBlockerStore defines the store interface needed for IP blocking.
type IPBlockerStore interface {
	IsIPBlocked(ipAddress string) (bool, error)
}

// NewIPBlocker returns a middleware that blocks requests from blacklisted IPs.
func NewIPBlocker(store IPBlockerStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := httputil.ClientIP(r)
			if ip == "" {
				// Cannot determine IP, allow through
				next.ServeHTTP(w, r)
				return
			}

			blocked, err := store.IsIPBlocked(ip)
			if err != nil {
				log.Printf("Error checking IP block status for %s: %v", ip, err)
				// On error, allow through (fail-open to prevent legitimate traffic blocking)
				next.ServeHTTP(w, r)
				return
			}

			if blocked {
				log.Printf("Blocked request from blacklisted IP: %s %s %s", ip, r.Method, r.URL.Path)
				metrics.IncrementBlockedIPRequests()
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
