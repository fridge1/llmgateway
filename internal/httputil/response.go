package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

func ExtractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}

func IsValidToken(token string, validTokens []string) bool {
	for _, t := range validTokens {
		if token == t {
			return true
		}
	}
	return false
}

func WriteError(w http.ResponseWriter, status int, message, errType, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"message": message,
			"type":    errType,
			"code":    code,
		},
	}); err != nil {
		slog.Error("failed to write error response", "error", err)
	}
}
