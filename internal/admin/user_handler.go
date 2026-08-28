package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// AuditLogger is an interface for recording admin audit events.
type AuditLogger interface {
	Log(r *http.Request, action, resourceType, resourceID string, details map[string]any)
}

// UserHandler provides admin user management endpoints.
type UserHandler struct {
	store store.Store
	audit AuditLogger
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(s store.Store, audit ...AuditLogger) *UserHandler {
	h := &UserHandler{store: s}
	if len(audit) > 0 {
		h.audit = audit[0]
	}
	return h
}

// HandleListUsers returns paginated list of all users.
// GET /api/admin/users?page=1&size=20
func (h *UserHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	users, filteredTotal, globalTotal, globalActiveCount, globalTotalBalance, err := h.store.ListUsersWithBalance(size, offset, search)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list users", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"users":         users,
		"total":         filteredTotal,
		"global_total":  globalTotal,
		"active_count":  globalActiveCount,
		"total_balance": globalTotalBalance,
		"page":          page,
		"size":          size,
	})
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

// HandleUpdateUserStatus updates a user's status.
// PUT /api/admin/users/{id}/status
func (h *UserHandler) HandleUpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path: /api/admin/users/{id}/status
	path := strings.TrimRight(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// Expected: ["", "api", "admin", "users", "{id}", "status"]
	if len(parts) < 6 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid path", "invalid_request_error", "bad_path")
		return
	}
	userID := parts[len(parts)-2]

	target, err := h.store.GetUserByID(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "user not found", "invalid_request_error", "not_found")
		return
	}
	if target.Role == "admin" {
		httputil.WriteError(w, http.StatusForbidden, "cannot modify admin user", "forbidden", "admin_protected")
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}

	if req.Status != "active" && req.Status != "disabled" {
		httputil.WriteError(w, http.StatusBadRequest, "status must be 'active' or 'disabled'", "invalid_request_error", "invalid_status")
		return
	}

	if err := h.store.UpdateUserStatus(userID, req.Status); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update user status", "server_error", "db_error")
		return
	}

	if h.audit != nil {
		h.audit.Log(r, "update_user_status", "user", userID, map[string]any{"status": req.Status})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleDeleteUser deletes a user and all associated data.
// DELETE /api/admin/users/{id}
func (h *UserHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// Expected: ["", "api", "admin", "users", "{id}"]
	if len(parts) < 5 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid path", "invalid_request_error", "bad_path")
		return
	}
	userID := parts[len(parts)-1]

	target, err := h.store.GetUserByID(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "user not found", "invalid_request_error", "not_found")
		return
	}
	if target.Role == "admin" {
		httputil.WriteError(w, http.StatusForbidden, "cannot delete admin user", "forbidden", "admin_protected")
		return
	}

	if err := h.store.DeleteUser(userID); err != nil {
		if strings.Contains(err.Error(), "user not found") {
			httputil.WriteError(w, http.StatusNotFound, "user not found", "invalid_request_error", "not_found")
			return
		}
		if strings.Contains(err.Error(), "user owns tenants") {
			httputil.WriteError(w, http.StatusConflict, "user owns tenants, transfer ownership before deletion", "conflict", "tenant_owner")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete user", "server_error", "db_error")
		return
	}

	if h.audit != nil {
		h.audit.Log(r, "delete_user", "user", userID, nil)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type rechargeRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// HandleRecharge adds balance to a user's account.
// POST /api/admin/users/{id}/recharge
func (h *UserHandler) HandleRecharge(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path: /api/admin/users/{id}/recharge
	path := strings.TrimRight(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// Expected: ["", "api", "admin", "users", "{id}", "recharge"]
	if len(parts) < 6 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid path", "invalid_request_error", "bad_path")
		return
	}
	userID := parts[len(parts)-2]

	var req rechargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}

	if req.Amount <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "amount must be positive", "invalid_request_error", "invalid_amount")
		return
	}

	if req.Amount > 100000 {
		httputil.WriteError(w, http.StatusBadRequest, "单次充值不能超过 100000 元", "invalid_request_error", "amount_too_large")
		return
	}

	if req.Description == "" {
		req.Description = "Admin recharge"
	}

	if err := h.store.Recharge(userID, req.Amount, req.Description); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to recharge", "server_error", "db_error")
		return
	}

	if h.audit != nil {
		h.audit.Log(r, "admin_recharge", "user", userID, map[string]any{"amount": req.Amount, "description": req.Description})
	}

	// Return the updated balance.
	balance, err := h.store.GetBalance(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "recharged but failed to get balance", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"balance": balance,
	})
}

// HandleDashboard returns aggregate stats for the admin dashboard.
// GET /api/admin/dashboard
func (h *UserHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetAdminDashboardStats()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get dashboard stats", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleConsumptionStats returns global token consumption and cost breakdown per model.
// GET /api/admin/consumption-stats?days=30
func (h *UserHandler) HandleConsumptionStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30
	}

	stats, err := h.store.GetAdminConsumptionStats(days)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get consumption stats", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleImageDurationStats returns per-model image generation duration stats.
// GET /api/admin/image-duration-stats?days=30
func (h *UserHandler) HandleImageDurationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30
	}

	stats, err := h.store.GetImageDurationStats(days)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get image duration stats", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"models": stats})
}

// funnel for users registered within the given window.
// GET /api/admin/funnel-stats?days=30
func (h *UserHandler) HandleFunnelStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30
	}

	stats, err := h.store.GetAdminFunnelStats(days)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get funnel stats", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleListAllOrders returns paginated orders across all users for admin.
// GET /api/admin/orders?page=1&size=20&status=paid
func (h *UserHandler) HandleListAllOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	status := r.URL.Query().Get("status")
	switch status {
	case "paid", "pending", "expired":
	default:
		status = ""
	}

	orders, total, err := h.store.ListAllOrders(size, offset, status)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list orders", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"orders": orders,
		"total":  total,
		"page":   page,
		"size":   size,
	})
}

// userIDFromPath extracts the {id} segment from /api/admin/users/{id}/...
func userIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if p == "users" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func parseDateRange(r *http.Request) (startDate, endDate *time.Time) {
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
	return startDate, endDate
}

// HandleUserTransactions returns a single user's paginated transactions.
// GET /api/admin/users/{id}/transactions?page=&size=&start_date=&end_date=&type=
func (h *UserHandler) HandleUserTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	userID := userIDFromPath(r.URL.Path)
	if userID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid path", "invalid_request_error", "bad_path")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	typeFilter := r.URL.Query().Get("type")
	switch typeFilter {
	case "consumption", "recharge", "subscription_usage", "sub_purchase", "refund":
	default:
		typeFilter = ""
	}

	startDate, endDate := parseDateRange(r)

	transactions, total, sums, err := h.store.ListTransactions(userID, size, offset, typeFilter, startDate, endDate)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list transactions", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"transactions": transactions,
		"total":        total,
		"sums":         sums,
		"page":         page,
		"size":         size,
	})
}

// HandleUserConsumptionStats returns a single user's per-model consumption and daily trend.
// GET /api/admin/users/{id}/consumption-stats?days=30
func (h *UserHandler) HandleUserConsumptionStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	userID := userIDFromPath(r.URL.Path)
	if userID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid path", "invalid_request_error", "bad_path")
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30
	}

	stats, err := h.store.GetUserConsumptionStats(userID, days)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get consumption stats", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleExportUserTransactions exports a single user's transactions as an Excel file.
// GET /api/admin/users/{id}/transactions/export?start_date=&end_date=
func (h *UserHandler) HandleExportUserTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	userID := userIDFromPath(r.URL.Path)
	if userID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid path", "invalid_request_error", "bad_path")
		return
	}

	startDate, endDate := parseDateRange(r)

	transactions, err := h.store.ListUserTransactionsForExport(userID, startDate, endDate)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to export", "server_error", "export_failed")
		return
	}

	f := excelize.NewFile()
	sheet := "Sheet1"

	headers := []string{"时间", "类型", "模型", "输入Token", "输出Token", "缓存命中", "缓存写入", "金额(元)"}
	for i, head := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, head)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "H1", headerStyle)

	cellName := func(col, row int) string {
		name, _ := excelize.CoordinatesToCellName(col, row)
		return name
	}

	for row, tx := range transactions {
		rIdx := row + 2
		f.SetCellValue(sheet, cellName(1, rIdx), tx.CreatedAt.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheet, cellName(2, rIdx), tx.Type)
		if tx.Model != nil {
			f.SetCellValue(sheet, cellName(3, rIdx), *tx.Model)
		}
		if tx.PromptTokens != nil {
			f.SetCellValue(sheet, cellName(4, rIdx), *tx.PromptTokens)
		}
		if tx.CompletionTokens != nil {
			f.SetCellValue(sheet, cellName(5, rIdx), *tx.CompletionTokens)
		}
		if tx.CacheReadTokens != nil {
			f.SetCellValue(sheet, cellName(6, rIdx), *tx.CacheReadTokens)
		}
		if tx.CacheCreationTokens != nil {
			f.SetCellValue(sheet, cellName(7, rIdx), *tx.CacheCreationTokens)
		}
		f.SetCellValue(sheet, cellName(8, rIdx), fmt.Sprintf("%.4f", tx.Amount))
	}

	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 16)
	}

	filename := fmt.Sprintf("user_%s_transactions_%s.xlsx", userID, time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	if err := f.Write(w); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to write excel", "server_error", "write_failed")
	}
}


type updateRoleRequestAdmin struct {
	Role string `json:"role"`
}

// HandleUpdateUserRole changes a user's platform role (admin only — the RBAC
// middleware allows support onto /api/admin/users, so the route dispatcher
// must gate this behind an explicit admin check).
// PUT /api/admin/users/{id}/role
func (h *UserHandler) HandleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// Expected: ["", "api", "admin", "users", "{id}", "role"]
	if len(parts) < 6 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid path", "invalid_request_error", "bad_path")
		return
	}
	userID := parts[len(parts)-2]

	callerID, _ := r.Context().Value(CtxUserIDKey).(string)
	if callerID == userID {
		httputil.WriteError(w, http.StatusBadRequest, "cannot change your own role", "invalid_request_error", "self_role_change")
		return
	}

	target, err := h.store.GetUserByID(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "user not found", "invalid_request_error", "not_found")
		return
	}
	if target.Role == "admin" {
		httputil.WriteError(w, http.StatusForbidden, "cannot modify admin user", "forbidden", "admin_protected")
		return
	}

	var req updateRoleRequestAdmin
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}
	if req.Role == "admin" || !store.ValidPlatformRole(req.Role) {
		httputil.WriteError(w, http.StatusBadRequest, "role must be one of: user, support, finance, ops", "invalid_request_error", "invalid_role")
		return
	}

	if err := h.store.UpdateUserRole(userID, req.Role); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update user role", "server_error", "db_error")
		return
	}

	if h.audit != nil {
		h.audit.Log(r, "update_user_role", "user", userID, map[string]any{"role": req.Role})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
