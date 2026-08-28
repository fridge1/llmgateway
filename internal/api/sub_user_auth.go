package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// SubUserAuthHandler handles sub-user authentication.
type SubUserAuthHandler struct {
	store     store.Store
	jwtSecret []byte
}

// NewSubUserAuthHandler creates a SubUserAuthHandler.
func NewSubUserAuthHandler(s store.Store, jwtSecret string) *SubUserAuthHandler {
	return &SubUserAuthHandler{store: s, jwtSecret: []byte(jwtSecret)}
}

type subUserLoginRequest struct {
	TenantID string `json:"tenant_id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type subUserLoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	TenantID string `json:"tenant_id"`
}

// HandleSubUserLogin authenticates a tenant sub-user and returns a JWT.
func (h *SubUserAuthHandler) HandleSubUserLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	var req subUserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.TenantID == "" || req.Username == "" || req.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "tenant_id, username and password are required", "invalid_request_error", "missing_fields")
		return
	}

	subUser, err := h.store.AuthenticateSubUser(req.TenantID, req.Username, req.Password)
	if err != nil {
		slog.Error("sub-user authentication failed", "tenant_id", req.TenantID, "username", req.Username, "error", err)
		httputil.WriteError(w, http.StatusUnauthorized, "用户名或密码错误", "auth_error", "invalid_credentials")
		return
	}

	if subUser.Status != "active" {
		httputil.WriteError(w, http.StatusForbidden, "账户已被禁用", "auth_error", "account_disabled")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":          subUser.ID,
		"tenant_id":    subUser.TenantID,
		"username":     subUser.Username,
		"role":         "sub_user",
		"exp":          time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString(h.jwtSecret)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to generate token", "server_error", "jwt_error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "sub_token",
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subUserLoginResponse{
		Token:    tokenStr,
		Username: subUser.Username,
		TenantID: subUser.TenantID,
	})
}

// HandleSubUserMe returns the current sub-user's info from JWT.
func (h *SubUserAuthHandler) HandleSubUserMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUserID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	role, _ := r.Context().Value(admin.CtxRoleKey).(string)
	if role != "sub_user" {
		httputil.WriteError(w, http.StatusForbidden, "not a sub-user", "auth_error", "invalid_role")
		return
	}

	subUser, err := h.store.GetTenantSubUser(subUserID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "sub-user not found", "invalid_request_error", "not_found")
		return
	}

	resp := map[string]any{
		"sub_user_id": subUser.ID,
		"tenant_id":   subUser.TenantID,
		"username":    subUser.Username,
		"nickname":    subUser.Nickname,
		"status":      subUser.Status,
		"quota_limit": subUser.QuotaLimit,
		"quota_used":  subUser.QuotaUsed,
		"role":        "sub_user",
	}

	if subUser.QuotaLimit != nil {
		remaining := *subUser.QuotaLimit - subUser.QuotaUsed
		resp["quota_remaining"] = remaining
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleSubUserLogout clears the auth cookie for sub-users.
func (h *SubUserAuthHandler) HandleSubUserLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "sub_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *SubUserAuthHandler) requireSubUser(r *http.Request) (string, error) {
	role, _ := r.Context().Value(admin.CtxRoleKey).(string)
	if role != "sub_user" {
		return "", fmt.Errorf("not a sub-user")
	}
	subUserID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	return subUserID, nil
}

// HandleSubUserListKeys lists the sub-user's own API keys.
func (h *SubUserAuthHandler) HandleSubUserListKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUserID, err := h.requireSubUser(r)
	if err != nil {
		httputil.WriteError(w, http.StatusForbidden, "not a sub-user", "auth_error", "invalid_role")
		return
	}

	keys, err := h.store.ListTenantSubUserKeys(subUserID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list keys", "api_error", "list_failed")
		return
	}
	if keys == nil {
		keys = []store.TenantSubUserKey{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

// HandleSubUserCreateKey creates a new API key for the sub-user.
func (h *SubUserAuthHandler) HandleSubUserCreateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUserID, err := h.requireSubUser(r)
	if err != nil {
		httputil.WriteError(w, http.StatusForbidden, "not a sub-user", "auth_error", "invalid_role")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	subUser, err := h.store.GetTenantSubUser(subUserID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "sub-user not found", "invalid_request_error", "not_found")
		return
	}

	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to generate key", "api_error", "keygen_failed")
		return
	}
	keyStr := "sk-sub-" + hex.EncodeToString(rawKey)
	keyHash := sha256.Sum256([]byte(keyStr))
	keyHashStr := hex.EncodeToString(keyHash[:])
	keyPrefix := keyStr[:12]

	key, err := h.store.CreateTenantSubUserKey(subUserID, subUser.TenantID, keyHashStr, keyPrefix, req.Name)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create key", "api_error", "create_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":         key.ID,
		"name":       key.Name,
		"key":        keyStr,
		"key_prefix": key.KeyPrefix,
		"created_at": key.CreatedAt,
	})
}

// HandleSubUserDeleteKey deletes a sub-user's API key.
func (h *SubUserAuthHandler) HandleSubUserDeleteKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUserID, err := h.requireSubUser(r)
	if err != nil {
		httputil.WriteError(w, http.StatusForbidden, "not a sub-user", "auth_error", "invalid_role")
		return
	}

	keyID := extractPathSegment(r.URL.Path, "keys")
	if keyID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "key id is required", "invalid_request_error", "missing_id")
		return
	}

	if err := h.store.DeleteSubUserOwnKey(keyID, subUserID); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "key not found", "invalid_request_error", "not_found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleSubUserTransactions lists the sub-user's own transactions.
func (h *SubUserAuthHandler) HandleSubUserTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUserID, err := h.requireSubUser(r)
	if err != nil {
		httputil.WriteError(w, http.StatusForbidden, "not a sub-user", "auth_error", "invalid_role")
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	transactions, total, err := h.store.ListTenantSubUserTransactions(subUserID, limit, offset)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list transactions", "api_error", "list_failed")
		return
	}
	if transactions == nil {
		transactions = []store.Transaction{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"transactions": transactions,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// HandleSubUserStats returns usage statistics for the sub-user.
func (h *SubUserAuthHandler) HandleSubUserStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUserID, err := h.requireSubUser(r)
	if err != nil {
		slog.Error("sub-user stats: requireSubUser failed", "error", err)
		httputil.WriteError(w, http.StatusForbidden, "not a sub-user", "auth_error", "invalid_role")
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 1 || days > 90 {
		days = 7
	}

	slog.Info("fetching sub-user stats", "sub_user_id", subUserID, "days", days)

	stats, err := h.store.GetSubUserBillingStats(subUserID, days)
	if err != nil {
		slog.Error("failed to get sub-user stats", "sub_user_id", subUserID, "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get stats", "api_error", "stats_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
