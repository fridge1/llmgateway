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
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// TenantSubUserHandler provides tenant sub-user management API endpoints.
type TenantSubUserHandler struct {
	store store.Store
}

// NewTenantSubUserHandler creates a TenantSubUserHandler.
func NewTenantSubUserHandler(s store.Store) *TenantSubUserHandler {
	return &TenantSubUserHandler{store: s}
}

// requireSubUserInTenant 校验 path 中的 subUserID 确实属于 path 中的 tenantID。
// 中间件只校验调用者是 path tenantID 的 owner/admin，但 subUserID 可能指向
// 其他租户的子用户，必须在 handler 层做归属校验，否则会形成跨租户越权。
// 返回 (subUser, true)；任何不匹配/查询失败都直接写 404 并返回 false。
func (h *TenantSubUserHandler) requireSubUserInTenant(w http.ResponseWriter, r *http.Request) (*store.TenantSubUser, string, bool) {
	tenantID := extractPathSegment(r.URL.Path, "tenants")
	subUserID := extractPathSegment(r.URL.Path, "sub-users")
	if tenantID == "" || subUserID == "" {
		httputil.WriteError(w, http.StatusNotFound, "sub-user not found", "invalid_request_error", "not_found")
		return nil, "", false
	}
	subUser, err := h.store.GetTenantSubUser(subUserID)
	if err != nil || subUser.TenantID != tenantID {
		httputil.WriteError(w, http.StatusNotFound, "sub-user not found", "invalid_request_error", "not_found")
		return nil, tenantID, false
	}
	return subUser, tenantID, true
}

// ---------------------------------------------------------------------------
// Sub-User CRUD
// ---------------------------------------------------------------------------

type createSubUserRequest struct {
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	Nickname   string   `json:"nickname"`
	QuotaLimit *float64 `json:"quota_limit"`
}

func (h *TenantSubUserHandler) HandleCreateSubUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)

	slog.Info("CreateSubUser called", "tenant_id", tenantID, "user_id", userID, "user_id_len", len(userID))

	var req createSubUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "username and password are required", "invalid_request_error", "missing_fields")
		return
	}

	subUser, err := h.store.CreateTenantSubUser(
		tenantID,
		strings.TrimSpace(req.Username),
		req.Password,
		strings.TrimSpace(req.Nickname),
		req.QuotaLimit,
		userID,
	)
	if err != nil {
		slog.Error("failed to create sub-user", "error", err, "tenant_id", tenantID, "username", req.Username)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create sub-user", "api_error", "create_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(subUser)
}

func (h *TenantSubUserHandler) HandleListSubUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	subUsers, err := h.store.ListTenantSubUsers(tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list sub-users", "api_error", "list_failed")
		return
	}
	if subUsers == nil {
		subUsers = []store.TenantSubUserWithQuota{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subUsers)
}

func (h *TenantSubUserHandler) HandleGetSubUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUser, _, ok := h.requireSubUserInTenant(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subUser)
}

type updateSubUserQuotaRequest struct {
	QuotaLimit *float64 `json:"quota_limit"`
}

func (h *TenantSubUserHandler) HandleUpdateSubUserQuota(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUser, _, ok := h.requireSubUserInTenant(w, r)
	if !ok {
		return
	}
	var req updateSubUserQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	if err := h.store.UpdateTenantSubUserQuota(subUser.ID, req.QuotaLimit); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update quota", "api_error", "update_failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type resetSubUserPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

func (h *TenantSubUserHandler) HandleResetSubUserPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUser, _, ok := h.requireSubUserInTenant(w, r)
	if !ok {
		return
	}
	var req resetSubUserPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if strings.TrimSpace(req.NewPassword) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "new_password is required", "invalid_request_error", "missing_password")
		return
	}

	if err := h.store.ResetTenantSubUserPassword(subUser.ID, req.NewPassword); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to reset password", "api_error", "reset_failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TenantSubUserHandler) HandleDeleteSubUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUser, _, ok := h.requireSubUserInTenant(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteTenantSubUser(subUser.ID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete sub-user", "api_error", "delete_failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Sub-User API Keys
// ---------------------------------------------------------------------------

type createSubUserKeyRequest struct {
	Name string `json:"name"`
}

func (h *TenantSubUserHandler) HandleCreateSubUserKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUser, tenantID, ok := h.requireSubUserInTenant(w, r)
	if !ok {
		return
	}
	var req createSubUserKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
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

	key, err := h.store.CreateTenantSubUserKey(subUser.ID, tenantID, keyHashStr, keyPrefix, req.Name)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create key", "api_error", "create_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":          key.ID,
		"name":        key.Name,
		"key":         keyStr,
		"key_prefix":  key.KeyPrefix,
		"created_at":  key.CreatedAt,
	})
}

func (h *TenantSubUserHandler) HandleListSubUserKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUser, _, ok := h.requireSubUserInTenant(w, r)
	if !ok {
		return
	}
	keys, err := h.store.ListTenantSubUserKeys(subUser.ID)
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

func (h *TenantSubUserHandler) HandleDeleteSubUserKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	keyID := extractPathSegment(r.URL.Path, "keys")
	if tenantID == "" || keyID == "" {
		httputil.WriteError(w, http.StatusNotFound, "key not found", "invalid_request_error", "not_found")
		return
	}
	if err := h.store.DeleteTenantSubUserKey(keyID, tenantID); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "key not found", "invalid_request_error", "not_found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Sub-User Transactions
// ---------------------------------------------------------------------------

func (h *TenantSubUserHandler) HandleListSubUserTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUser, _, ok := h.requireSubUserInTenant(w, r)
	if !ok {
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

	transactions, total, err := h.store.ListTenantSubUserTransactions(subUser.ID, limit, offset)
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

func (h *TenantSubUserHandler) HandleListAllSubUserTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	subUserID := r.URL.Query().Get("sub_user_id")
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

	transactions, total, err := h.store.ListTenantAllSubUserTransactions(tenantID, limit, offset, subUserID)
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

// ---------------------------------------------------------------------------
// Tenant Stats & Export
// ---------------------------------------------------------------------------

func (h *TenantSubUserHandler) HandleTenantStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	days := 7
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}

	stats, err := h.store.GetTenantBillingStats(tenantID, days)
	if err != nil {
		slog.Error("failed to get tenant stats", "error", err, "tenant_id", tenantID)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get stats", "api_error", "stats_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *TenantSubUserHandler) HandleExportTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	subUserID := r.URL.Query().Get("sub_user_id")

	var startDate, endDate *time.Time
	if v := r.URL.Query().Get("start_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			startDate = &t
		}
	}
	if v := r.URL.Query().Get("end_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(24*time.Hour - time.Second)
			endDate = &end
		}
	}

	transactions, err := h.store.ListTenantTransactionsForExport(tenantID, startDate, endDate, subUserID)
	if err != nil {
		slog.Error("failed to list transactions for export", "error", err, "tenant_id", tenantID)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to export", "api_error", "export_failed")
		return
	}

	f := excelize.NewFile()
	sheet := "Sheet1"

	headers := []string{"时间", "子用户", "模型", "输入Token", "输出Token", "缓存命中", "缓存写入", "金额(元)"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "H1", headerStyle)

	for row, tx := range transactions {
		r := row + 2
		f.SetCellValue(sheet, cellName(1, r), tx.CreatedAt.Format("2006-01-02 15:04:05"))
		if tx.SubUserUsername != nil {
			f.SetCellValue(sheet, cellName(2, r), *tx.SubUserUsername)
		}
		if tx.Model != nil {
			f.SetCellValue(sheet, cellName(3, r), *tx.Model)
		}
		if tx.PromptTokens != nil {
			f.SetCellValue(sheet, cellName(4, r), *tx.PromptTokens)
		}
		if tx.CompletionTokens != nil {
			f.SetCellValue(sheet, cellName(5, r), *tx.CompletionTokens)
		}
		if tx.CacheReadTokens != nil {
			f.SetCellValue(sheet, cellName(6, r), *tx.CacheReadTokens)
		}
		if tx.CacheCreationTokens != nil {
			f.SetCellValue(sheet, cellName(7, r), *tx.CacheCreationTokens)
		}
		f.SetCellValue(sheet, cellName(8, r), fmt.Sprintf("%.4f", -tx.Amount))
	}

	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 15)
	}

	filename := fmt.Sprintf("transactions_%s.xlsx", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	if err := f.Write(w); err != nil {
		slog.Error("failed to write excel", "error", err)
	}
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func extractPathSegment(path, prefix string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if p == prefix && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// HandleGetSubUserModelStats returns model-level statistics for a sub-user.
// GET /api/tenants/{tenant_id}/sub-users/{sub_user_id}/model-stats
func (h *TenantSubUserHandler) HandleGetSubUserModelStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	subUser, _, ok := h.requireSubUserInTenant(w, r)
	if !ok {
		return
	}

	// Parse date range parameters
	var startDate, endDate *time.Time
	if v := r.URL.Query().Get("start_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			startDate = &t
		} else {
			httputil.WriteError(w, http.StatusBadRequest, "invalid start_date format, expected YYYY-MM-DD", "invalid_request_error", "invalid_date")
			return
		}
	}
	if v := r.URL.Query().Get("end_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			// Include the entire end date by setting time to 23:59:59
			end := t.Add(24*time.Hour - time.Second)
			endDate = &end
		} else {
			httputil.WriteError(w, http.StatusBadRequest, "invalid end_date format, expected YYYY-MM-DD", "invalid_request_error", "invalid_date")
			return
		}
	}

	stats, err := h.store.GetSubUserModelStats(subUser.ID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get sub-user model stats", "error", err, "sub_user_id", subUser.ID)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get model statistics", "api_error", "stats_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

