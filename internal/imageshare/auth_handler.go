package imageshare

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
)

// AuthHandler handles image-share login/logout/me.
type AuthHandler struct {
	store     *Store
	jwtSecret []byte
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(s *Store, jwtSecret string) *AuthHandler {
	return &AuthHandler{store: s, jwtSecret: []byte(jwtSecret)}
}

type loginRequest struct {
	Key string `json:"key"`
}

type loginResponse struct {
	Token       string `json:"token"`
	KeyID       string `json:"key_id"`
	Name        string `json:"name"`
	QuotaTotal  int    `json:"quota_total"`
	QuotaUsed   int    `json:"quota_used"`
	OwnerUserID string `json:"owner_user_id"`
}

// HandleLogin authenticates a plain image-share key and sets the image_token cookie.
// POST /api/image-share/login
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	hash := HashKey(req.Key)
	key, err := h.store.GetKeyByHash(hash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteError(w, http.StatusUnauthorized, "密钥无效", "auth_error", "invalid_key")
			return
		}
		slog.Error("imageshare login: lookup failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal error", "server_error", "db_error")
		return
	}
	if key.Status != "active" {
		httputil.WriteError(w, http.StatusForbidden, "密钥已禁用", "auth_error", "key_disabled")
		return
	}
	enabled, status, err := h.store.IsImageShareEnabled(key.OwnerUserID)
	if err != nil || !enabled || status != "active" {
		httputil.WriteError(w, http.StatusForbidden, "密钥不可用", "auth_error", "owner_disabled")
		return
	}
	if key.QuotaUsed >= key.QuotaTotal {
		httputil.WriteError(w, http.StatusForbidden, "配额已用尽", "auth_error", "quota_exhausted")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":           key.ID,
		"role":          RoleImageShare,
		"owner_user_id": key.OwnerUserID,
		"exp":           time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString(h.jwtSecret)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to sign token", "server_error", "jwt_error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{
		Token:       tokenStr,
		KeyID:       key.ID,
		Name:        key.Name,
		QuotaTotal:  key.QuotaTotal,
		QuotaUsed:   key.QuotaUsed,
		OwnerUserID: key.OwnerUserID,
	})
}

// HandleMe returns current image-share session info, with fresh quota numbers.
// GET /api/image-share/me
func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value(admin.CtxRoleKey).(string)
	if role != RoleImageShare {
		httputil.WriteError(w, http.StatusForbidden, "not an image-share session", "auth_error", "wrong_role")
		return
	}
	keyID := KeyIDFromContext(r.Context())
	key, err := h.store.GetKeyByID(keyID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "key not found", "invalid_request_error", "not_found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"role":            RoleImageShare,
		"key_id":          key.ID,
		"name":            key.Name,
		"key_prefix":      key.KeyPrefix,
		"quota_total":     key.QuotaTotal,
		"quota_used":      key.QuotaUsed,
		"quota_remaining": key.Remaining(),
		"owner_user_id":   key.OwnerUserID,
		"status":          key.Status,
	})
}

// HandleLogout clears the image_token cookie.
// POST /api/image-share/logout
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
