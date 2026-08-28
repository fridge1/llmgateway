package imageshare

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
)

// KeysHandler manages parent-user CRUD over their image-share keys.
type KeysHandler struct {
	store *Store
}

// NewKeysHandler returns a KeysHandler.
func NewKeysHandler(s *Store) *KeysHandler {
	return &KeysHandler{store: s}
}

// requireOwner returns the JWT user id for a regular logged-in user, after ensuring
// image_share_enabled is true on their account. image_share role is rejected here.
func (h *KeysHandler) requireOwner(w http.ResponseWriter, r *http.Request) (string, bool) {
	role, _ := r.Context().Value(admin.CtxRoleKey).(string)
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" || role == RoleImageShare {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return "", false
	}
	enabled, status, err := h.store.IsImageShareEnabled(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to check permission", "server_error", "db_error")
		return "", false
	}
	if !enabled || status != "active" {
		httputil.WriteError(w, http.StatusForbidden, "未开通图片分发权限", "auth_error", "feature_disabled")
		return "", false
	}
	return userID, true
}

type createKeyRequest struct {
	Name       string `json:"name"`
	QuotaTotal int    `json:"quota_total"`
}

type createKeyResponse struct {
	Key string `json:"key"` // plain key, shown only once
	Key2
}

// Key2 is a JSON-friendly subset (avoid leaking key_hash).
type Key2 struct {
	ID         string `json:"id"`
	OwnerID    string `json:"owner_user_id"`
	KeyPrefix  string `json:"key_prefix"`
	Name       string `json:"name"`
	QuotaTotal int    `json:"quota_total"`
	QuotaUsed  int    `json:"quota_used"`
	Status     string `json:"status"`
}

func toKey2(k *Key) Key2 {
	return Key2{
		ID:         k.ID,
		OwnerID:    k.OwnerUserID,
		KeyPrefix:  k.KeyPrefix,
		Name:       k.Name,
		QuotaTotal: k.QuotaTotal,
		QuotaUsed:  k.QuotaUsed,
		Status:     k.Status,
	}
}

// HandleList returns all keys owned by the caller.
// GET /api/image-share/keys
func (h *KeysHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	ownerID, ok := h.requireOwner(w, r)
	if !ok {
		return
	}
	keys, err := h.store.ListKeysByOwner(ownerID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list keys", "server_error", "db_error")
		return
	}
	out := make([]Key, 0, len(keys))
	for i := range keys {
		// hide hash explicitly via json:"-" tag — already on struct
		out = append(out, keys[i])
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"keys": out})
}

// HandleCreate creates a new key.
// POST /api/image-share/keys
func (h *KeysHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	ownerID, ok := h.requireOwner(w, r)
	if !ok {
		return
	}
	var req createKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.QuotaTotal <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "quota_total must be > 0", "invalid_request_error", "bad_quota")
		return
	}
	if req.Name == "" {
		req.Name = "未命名"
	}
	plain, hash, prefix, err := GenerateKey()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to generate key", "server_error", "keygen_error")
		return
	}
	k, err := h.store.CreateKey(ownerID, hash, prefix, req.Name, req.QuotaTotal)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create key", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"key":         plain,
		"id":          k.ID,
		"key_prefix":  k.KeyPrefix,
		"name":        k.Name,
		"quota_total": k.QuotaTotal,
		"quota_used":  k.QuotaUsed,
		"status":      k.Status,
		"created_at":  k.CreatedAt,
	})
}

type patchKeyRequest struct {
	Name       *string `json:"name,omitempty"`
	Status     *string `json:"status,omitempty"`
	QuotaTotal *int    `json:"quota_total,omitempty"`
	ResetUsed  bool    `json:"reset_used,omitempty"`
}

// HandlePatch updates a key.
// PATCH /api/image-share/keys/{id}
func (h *KeysHandler) HandlePatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	ownerID, ok := h.requireOwner(w, r)
	if !ok {
		return
	}
	id := extractIDFromPath(r.URL.Path, "/api/image-share/keys/")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing key id", "invalid_request_error", "missing_id")
		return
	}
	var req patchKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	k, err := h.store.UpdateKey(id, ownerID, UpdatePatch{
		Name:       req.Name,
		Status:     req.Status,
		QuotaTotal: req.QuotaTotal,
		ResetUsed:  req.ResetUsed,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "key not found", "invalid_request_error", "not_found")
			return
		}
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "update_failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(k)
}

// HandleDelete removes a key.
// DELETE /api/image-share/keys/{id}
func (h *KeysHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	ownerID, ok := h.requireOwner(w, r)
	if !ok {
		return
	}
	id := extractIDFromPath(r.URL.Path, "/api/image-share/keys/")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing key id", "invalid_request_error", "missing_id")
		return
	}
	if err := h.store.DeleteKey(id, ownerID); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "key not found", "invalid_request_error", "not_found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete key", "server_error", "db_error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleKeysRouter dispatches PATCH/DELETE/GET for /api/image-share/keys/{id} and the
// collection endpoint /api/image-share/keys (GET list, POST create).
func (h *KeysHandler) HandleKeysRouter(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/image-share/keys")
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		switch r.Method {
		case http.MethodGet:
			h.HandleList(w, r)
		case http.MethodPost:
			h.HandleCreate(w, r)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		}
		return
	}
	switch r.Method {
	case http.MethodPatch:
		h.HandlePatch(w, r)
	case http.MethodDelete:
		h.HandleDelete(w, r)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
	}
}

// extractIDFromPath returns the path segment immediately after `prefix`.
func extractIDFromPath(path, prefix string) string {
	rest := strings.TrimPrefix(path, prefix)
	if rest == path {
		return ""
	}
	if i := strings.Index(rest, "/"); i >= 0 {
		return rest[:i]
	}
	return rest
}

// AdminHandler manages the admin-side image-share-enabled toggle.
type AdminHandler struct {
	store *Store
}

// NewAdminHandler returns an AdminHandler.
func NewAdminHandler(s *Store) *AdminHandler {
	return &AdminHandler{store: s}
}

type adminToggleRequest struct {
	Enabled bool `json:"enabled"`
}

// HandleToggle sets users.image_share_enabled. Caller must be authenticated as admin
// (enforced by middleware in main.go).
// PATCH /api/admin/users/{id}/image-share
func (h *AdminHandler) HandleToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	// Path: /api/admin/users/{id}/image-share
	rest := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	id := strings.TrimSuffix(rest, "/image-share")
	if id == "" || id == rest {
		httputil.WriteError(w, http.StatusBadRequest, "missing user id", "invalid_request_error", "missing_id")
		return
	}
	var req adminToggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if err := h.store.SetImageShareEnabled(id, req.Enabled); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "user not found", "invalid_request_error", "not_found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok", "user_id": id, "image_share_enabled": req.Enabled})
}
