package middleware

import (
	"log/slog"
	"net/http"
)

// CORS returns a middleware that adds Cross-Origin Resource Sharing headers.
// allowedOrigins specifies the allowed origins; if empty, behaviour depends on mode.
// mode "strict": empty allowedOrigins rejects cross-origin requests.
// mode "permissive" (default): empty allowedOrigins allows all ("*").
func CORS(allowedOrigins []string, mode string) func(http.Handler) http.Handler {
	// Build a lookup set for fast matching.
	if mode == "" {
		mode = "strict"
	}
	strict := mode == "strict"
	allowAll := len(allowedOrigins) == 0 && !strict
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
		}
		originSet[o] = true
	}

	if strict && len(allowedOrigins) == 0 {
		slog.Warn("CORS strict mode enabled with empty cors_origins — all cross-origin requests will be rejected")
	}

	// Security warning: allowing all origins with credentials is a security risk
	if allowAll {
		slog.Warn("CORS configured to allow all origins (*) with credentials=true — this is a security risk in production")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				// Note: Credentials are not set when allowing all origins
				// Browsers block credentials with wildcard origin for security
			} else if origin != "" && originSet[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				// Only set credentials for whitelisted origins
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			} else if strict && origin != "" {
				// Strict mode: reject cross-origin requests from non-whitelisted origins.
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				// For non-preflight, don't set CORS headers — browser will block the response.
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Api-Key")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight requests.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
