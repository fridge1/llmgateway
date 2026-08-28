package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/store"
)

// BlockedIPStore defines the store interface for IP blocking.
type BlockedIPStore interface {
	BlockIP(ipAddress, reason, blockedBy string, expiresAt *time.Time) error
	UnblockIP(ipAddress string) error
	GetBlockedIP(ipAddress string) (*store.BlockedIP, error)
	ListBlockedIPs(limit, offset int) ([]store.BlockedIP, error)
}

// BlockIPRequest represents a request to block an IP.
type BlockIPRequest struct {
	IPAddress     string  `json:"ip_address"`
	Reason        string  `json:"reason"`
	ExpiresInDays *int    `json:"expires_in_days"` // null = permanent
	Notes         *string `json:"notes"`
}

// BlockIPHandler blocks an IP address.
func BlockIPHandler(s BlockedIPStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BlockIPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.IPAddress == "" {
			http.Error(w, "ip_address is required", http.StatusBadRequest)
			return
		}

		if req.Reason == "" {
			http.Error(w, "reason is required", http.StatusBadRequest)
			return
		}

		// Calculate expiry time
		var expiresAt *time.Time
		if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
			t := time.Now().Add(time.Duration(*req.ExpiresInDays) * 24 * time.Hour)
			expiresAt = &t
		}

		// Get admin user from context
		var blockedBy string
		if userID, ok := r.Context().Value(CtxUserIDKey).(string); ok && userID != "" {
			blockedBy = userID
		}

		if err := s.BlockIP(req.IPAddress, req.Reason, blockedBy, expiresAt); err != nil {
			http.Error(w, "failed to block IP", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success":    true,
			"ip_address": req.IPAddress,
			"message":    "IP blocked successfully",
		})
	}
}

// UnblockIPHandler removes an IP from the blocklist.
func UnblockIPHandler(s BlockedIPStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract IP from path: /api/admin/blocked-ips/{ip}
		path := r.URL.Path
		ipAddress := strings.TrimPrefix(path, "/api/admin/blocked-ips/")
		if ipAddress == "" || ipAddress == path {
			http.Error(w, "ip_address is required", http.StatusBadRequest)
			return
		}

		if err := s.UnblockIP(ipAddress); err != nil {
			http.Error(w, "failed to unblock IP", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success":    true,
			"ip_address": ipAddress,
			"message":    "IP unblocked successfully",
		})
	}
}

// GetBlockedIPHandler retrieves a single blocked IP entry.
func GetBlockedIPHandler(s BlockedIPStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract IP from path: /api/admin/blocked-ips/{ip}
		path := r.URL.Path
		ipAddress := strings.TrimPrefix(path, "/api/admin/blocked-ips/")
		if ipAddress == "" || ipAddress == path {
			http.Error(w, "ip_address is required", http.StatusBadRequest)
			return
		}

		ip, err := s.GetBlockedIP(ipAddress)
		if err != nil {
			http.Error(w, "failed to get blocked IP", http.StatusInternalServerError)
			return
		}

		if ip == nil {
			http.Error(w, "IP not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ip)
	}
}

// ListBlockedIPsHandler lists all blocked IPs with pagination.
func ListBlockedIPsHandler(s BlockedIPStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 50
		}
		if limit > 500 {
			limit = 500
		}

		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if offset < 0 {
			offset = 0
		}

		ips, err := s.ListBlockedIPs(limit, offset)
		if err != nil {
			http.Error(w, "failed to list blocked IPs", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"items":  ips,
			"limit":  limit,
			"offset": offset,
		})
	}
}
