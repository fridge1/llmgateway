package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler manages API key CRUD operations.
type Handler struct {
	store store.Store
	cache *Cache
}

// NewHandler creates a new API key handler.
func NewHandler(s store.Store, cache *Cache) *Handler {
	return &Handler{store: s, cache: cache}
}

// GenerateAPIKey generates a new API key and returns (plainKey, hash, prefix).
func GenerateAPIKey() (string, string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("generate api key: %w", err)
	}
	plainKey := "sk-" + hex.EncodeToString(b)
	hash := HashAPIKey(plainKey)
	prefix := plainKey[:10] // "sk-" + first 7 hex chars
	return plainKey, hash, prefix, nil
}

// HashAPIKey returns the SHA-256 hex hash of an API key.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

type createKeyRequest struct {
	Name   string `json:"name"`
	PlanID *int   `json:"plan_id"`
}

type createKeyResponse struct {
	Key    string       `json:"key"` // plain key, shown only once
	APIKey store.APIKey `json:"api_key"`
}

// HandleList returns all API keys for the authenticated user.
// GET /api/keys
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	keys, err := h.store.ListAPIKeysByUser(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list keys", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"keys": keys})
}

// HandleCreate creates a new API key for the authenticated user.
// POST /api/keys
func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	var req createKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.Name == "" {
		req.Name = "default"
	}

	if req.PlanID != nil {
		subs, err := h.store.GetActiveSubscriptions(userID)
		if err != nil || !hasPlan(subs, *req.PlanID) {
			httputil.WriteError(w, http.StatusBadRequest, "该套餐无有效订阅", "invalid_request_error", "no_active_subscription")
			return
		}
	}

	plainKey, hash, prefix, err := GenerateAPIKey()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to generate key", "server_error", "keygen_error")
		return
	}

	apiKey, err := h.store.CreateAPIKey(userID, hash, prefix, req.Name, req.PlanID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create key", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createKeyResponse{Key: plainKey, APIKey: *apiKey})
}

// HandleRevoke deletes (revokes) an API key for the authenticated user.
// DELETE /api/keys/{id}
func (h *Handler) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	// Extract key ID from URL path: /api/keys/{id}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/keys/"), "/")
	keyID := parts[0]
	if keyID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing key id", "invalid_request_error", "missing_id")
		return
	}

	if err := h.store.DeleteAPIKey(keyID, userID); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "key not found", "invalid_request_error", "not_found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// HandleRevokeAll deletes all API keys for the authenticated user.
// POST /api/keys/revoke-all
func (h *Handler) HandleRevokeAll(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	count, err := h.store.RevokeAllAPIKeys(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to revoke keys", "server_error", "db_error")
		return
	}

	if h.cache != nil {
		h.cache.InvalidateUser(userID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok", "revoked": count})
}

// HandleAdminRevokeAllKeys deletes all API keys for a specified user (admin only).
// POST /api/admin/users/{id}/revoke-keys
func (h *Handler) HandleAdminRevokeAllKeys(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL path: /api/admin/users/{id}/revoke-keys
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	userID := strings.TrimSuffix(path, "/revoke-keys")
	if userID == "" || userID == path {
		httputil.WriteError(w, http.StatusBadRequest, "missing user id", "invalid_request_error", "missing_id")
		return
	}

	count, err := h.store.RevokeAllAPIKeys(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to revoke keys", "server_error", "db_error")
		return
	}

	if h.cache != nil {
		h.cache.InvalidateUser(userID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok", "revoked": count})
}

func hasPlan(subs []store.UserSubscription, planID int) bool {
	for _, s := range subs {
		if s.PlanID == planID {
			return true
		}
	}
	return false
}
