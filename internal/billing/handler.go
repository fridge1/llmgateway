package billing

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/export"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// BillingHandler provides billing API endpoints.
type BillingHandler struct {
	store store.Store
}

// NewBillingHandler creates a BillingHandler.
func NewBillingHandler(s store.Store) *BillingHandler {
	return &BillingHandler{store: s}
}

// HandleGetBalance returns the current user's balance.
// GET /api/billing/balance
func (h *BillingHandler) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	role, _ := r.Context().Value(admin.CtxRoleKey).(string)
	if role == "sub_user" {
		subUser, err := h.store.GetTenantSubUser(userID)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "sub-user not found", "not_found", "sub_user_not_found")
			return
		}
		resp := map[string]any{
			"sub_user_id": subUser.ID,
			"tenant_id":   subUser.TenantID,
			"username":    subUser.Username,
			"quota_limit": subUser.QuotaLimit,
			"quota_used":  subUser.QuotaUsed,
			"role":        "sub_user",
		}
		if subUser.QuotaLimit != nil {
			resp["quota_remaining"] = *subUser.QuotaLimit - subUser.QuotaUsed
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	balance, err := h.store.GetBalance(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get balance", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

// HandleListTransactions returns paginated transactions for the current user.
// GET /api/billing/transactions?page=1&size=20&type=consumption|recharge|subscription_usage
func (h *BillingHandler) HandleListTransactions(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
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
	if typeFilter != "" && typeFilter != "consumption" && typeFilter != "recharge" && typeFilter != "subscription_usage" && typeFilter != "sub_purchase" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid type filter, must be one of: consumption, recharge, subscription_usage, sub_purchase", "validation_error", "invalid_type")
		return
	}

	var startDate, endDate *time.Time
	if sd := r.URL.Query().Get("start_date"); sd != "" {
		t, err := time.Parse("2006-01-02", sd)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid start_date format, use YYYY-MM-DD", "validation_error", "invalid_start_date")
			return
		}
		startDate = &t
	}
	if ed := r.URL.Query().Get("end_date"); ed != "" {
		t, err := time.Parse("2006-01-02", ed)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid end_date format, use YYYY-MM-DD", "validation_error", "invalid_end_date")
			return
		}
		endOfDay := t.Add(24*time.Hour - time.Second)
		endDate = &endOfDay
	}

	txns, total, sums, err := h.store.ListTransactions(userID, size, offset, typeFilter, startDate, endDate)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list transactions", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"transactions":              txns,
		"total":                     total,
		"total_consumption":         sums.TotalConsumption,
		"total_recharge":            sums.TotalRecharge,
		"total_subscription_usage":  sums.TotalSubscriptionUsage,
		"total_sub_purchase": sums.TotalSubscriptionPurchase,
		"page":                      page,
		"size":                      size,
	})
}

// HandleGetStats returns billing statistics for the current user.
// GET /api/billing/stats?days=7
func (h *BillingHandler) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 1 || days > 90 {
		days = 7
	}

	stats, err := h.store.GetBillingStats(userID, days)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get billing stats", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleGetTokenStats returns all-time and period token aggregates for the current user.
// GET /api/billing/token-stats?days=7
func (h *BillingHandler) HandleGetTokenStats(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days != 7 && days != 30 && days != 90 {
		days = 7
	}

	stats, err := h.store.GetUserTokenStats(userID, days)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get token stats", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleGetKeyUsage returns per-key usage statistics for the current user.
// GET /api/billing/key-usage?days=30
func (h *BillingHandler) HandleGetKeyUsage(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 0 || days > 365 {
		days = 30
	}

	summaries, err := h.store.GetAPIKeyUsageSummary(userID, days)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get key usage", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

// HandleGetKeyTransactions returns paginated transactions for a specific API key.
// GET /api/billing/key-usage/{keyID}/transactions?page=1&size=20
func (h *BillingHandler) HandleGetKeyTransactions(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	keyID := r.PathValue("keyID")
	if keyID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing key ID", "validation_error", "missing_key_id")
		return
	}

	// Verify the key belongs to the user
	key, err := h.store.GetAPIKeyByID(keyID)
	if err != nil || key.UserID != userID {
		httputil.WriteError(w, http.StatusNotFound, "key not found", "not_found", "key_not_found")
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

	txns, total, err := h.store.ListTransactionsByAPIKey(keyID, size, offset)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list key transactions", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"transactions": txns,
		"total":        total,
		"page":         page,
		"size":         size,
	})
}


// HandleExportTransactions exports the current user's transactions as xlsx.
// GET /api/billing/transactions/export?start_date=&end_date=
func (h *BillingHandler) HandleExportTransactions(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	var startDate, endDate *time.Time
	if sd := r.URL.Query().Get("start_date"); sd != "" {
		if t, err := time.Parse("2006-01-02", sd); err == nil {
			startDate = &t
		}
	}
	if ed := r.URL.Query().Get("end_date"); ed != "" {
		if t, err := time.Parse("2006-01-02", ed); err == nil {
			end := t.Add(24*time.Hour - time.Second)
			endDate = &end
		}
	}

	txns, err := h.store.ListUserTransactionsForExport(userID, startDate, endDate)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to export", "server_error", "export_failed")
		return
	}
	if err := export.WriteTransactionsXLSX(w, "transactions", txns); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to write excel", "server_error", "write_failed")
	}
}
