package subscription

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/money"
	"github.com/zhulang/llm-gateway/internal/store"
)

// computeShortfall determines how much the user must pay to afford a plan.
// It returns the cent-rounded shortfall and whether the available balance is
// already sufficient (within money.EpsilonCNY tolerance). The shortfall is
// always rounded UP so the paid amount fully covers the gap — %.2f rounding
// down used to leave balances a fraction of a cent short, making the
// post-payment deduction fail probabilistically. A non-sufficient shortfall
// is floored at 0.01 (minimum order amount accepted by Alipay and CreateOrderWithPlan).
func computeShortfall(price, available float64) (shortfall float64, sufficient bool) {
	if available < 0 {
		available = 0
	}
	if money.GTE(available, price) {
		return 0, true
	}
	shortfall = money.Ceil2(price - available)
	if shortfall < 0.01 {
		shortfall = 0.01
	}
	return shortfall, false
}

// PaymentCreator abstracts payment URL creation for subscription shortfall payments.
type PaymentCreator interface {
	CreatePagePay(orderNo string, amount string, subject string) (string, error)
	CreateWapPay(orderNo string, amount string, subject string) (string, error)
}

// Handler provides subscription API endpoints.
type Handler struct {
	service *Service
	store   store.Store
	payment PaymentCreator
}

// NewHandler creates a subscription handler.
func NewHandler(svc *Service, s store.Store) *Handler {
	return &Handler{service: svc, store: s}
}

// SetPaymentCreator sets the payment creator for subscription shortfall payments.
func (h *Handler) SetPaymentCreator(p PaymentCreator) {
	h.payment = p
}

// HandleListPlans returns all available subscription plans.
// When the caller's tenant has custom pricing, purchase is disabled.
// GET /api/subscription/plans
func (h *Handler) HandleListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.store.ListSubscriptionPlans()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list plans", "server_error", "db_error")
		return
	}

	purchaseDisabled := false
	disabledReason := ""
	if userID, _ := r.Context().Value(admin.CtxUserIDKey).(string); userID != "" {
		if tenantID, terr := h.store.GetUserPrimaryPricingTenant(userID); terr == nil && tenantID != "" {
			purchaseDisabled = true
			disabledReason = "您所在的租户已配置专属定价，无法购买套餐"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"plans":             plans,
		"purchase_disabled": purchaseDisabled,
		"disabled_reason":   disabledReason,
	})
}

// HandleGetCurrent returns the user's current subscriptions and usage.
// GET /api/subscription/current
func (h *Handler) HandleGetCurrent(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "not authenticated", "auth_error", "no_user")
		return
	}

	subs, err := h.service.GetAllActiveSubscriptions(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get subscriptions", "server_error", "db_error")
		return
	}

	type subWithUsage struct {
		Subscription store.UserSubscription        `json:"subscription"`
		Usage        *store.SubscriptionUsageSummary `json:"usage"`
	}

	var items []subWithUsage
	for _, sub := range subs {
		usage, _ := h.service.GetUsageSummaryForSubscription(&sub)
		items = append(items, subWithUsage{Subscription: sub, Usage: usage})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"subscriptions": items,
	})
}

// HandleListHistory returns the user's full subscription history (active and historical).
// GET /api/subscription/history
func (h *Handler) HandleListHistory(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "not authenticated", "auth_error", "no_user")
		return
	}

	subs, err := h.store.ListUserSubscriptionHistory(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list subscription history", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": subs})
}

// HandleGetUsage returns the current period's usage details.
// GET /api/subscription/usage
func (h *Handler) HandleGetUsage(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "not authenticated", "auth_error", "no_user")
		return
	}

	usage, err := h.service.GetUsageSummary(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get usage", "server_error", "db_error")
		return
	}
	if usage == nil {
		httputil.WriteError(w, http.StatusNotFound, "no active subscription", "not_found", "no_subscription")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(usage)
}

type subscribeRequest struct {
	PlanID     int    `json:"plan_id"`
	ClientType string `json:"client_type"`
}

// HandleSubscribe creates a new subscription.
// If balance is sufficient, deducts directly. If not, creates a payment order for the shortfall.
// POST /api/subscription/subscribe
func (h *Handler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "not authenticated", "auth_error", "no_user")
		return
	}

	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}
	if req.PlanID <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "plan_id is required", "invalid_request", "missing_plan_id")
		return
	}

	sub, err := h.service.Subscribe(userID, req.PlanID)
	if err != nil {
		if !strings.Contains(err.Error(), "余额不足") || h.payment == nil {
			httputil.WriteError(w, http.StatusBadRequest, err.Error(), "subscription_error", "subscribe_failed")
			return
		}

		// Balance insufficient — return shortfall info only, do NOT create order yet.
		plan, planErr := h.store.GetSubscriptionPlan(req.PlanID)
		if planErr != nil {
			httputil.WriteError(w, http.StatusBadRequest, "plan not found", "subscription_error", "plan_not_found")
			return
		}

		bal, balErr := h.store.GetBalance(userID)
		if balErr != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to get balance", "server_error", "db_error")
			return
		}

		available := bal.Balance - bal.Frozen
		if available < 0 {
			available = 0
		}
		// Ceil2-rounded shortfall so the displayed amount matches what
		// create-payment will actually charge.
		shortfall, _ := computeShortfall(plan.MonthlyPriceCNY, available)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"need_payment": true,
			"shortfall":    shortfall,
			"balance":      available,
			"plan_price":   plan.MonthlyPriceCNY,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

// HandleCreateSubscriptionPayment creates a payment order for subscription shortfall.
// POST /api/subscription/create-payment
func (h *Handler) HandleCreateSubscriptionPayment(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "not authenticated", "auth_error", "no_user")
		return
	}

	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}
	if req.PlanID <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "plan_id is required", "invalid_request", "missing_plan_id")
		return
	}
	if h.payment == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "payment not configured", "server_error", "no_payment")
		return
	}

	plan, err := h.store.GetSubscriptionPlan(req.PlanID)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "plan not found", "subscription_error", "plan_not_found")
		return
	}

	bal, err := h.store.GetBalance(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get balance", "server_error", "db_error")
		return
	}

	available := bal.Balance - bal.Frozen
	shortfall, sufficient := computeShortfall(plan.MonthlyPriceCNY, available)
	if sufficient {
		httputil.WriteError(w, http.StatusBadRequest, "balance is sufficient, no payment needed", "subscription_error", "no_shortfall")
		return
	}

	planID := req.PlanID
	order, err := h.store.CreateOrderWithPlan(userID, shortfall, nil, &planID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create payment order", "server_error", "db_error")
		return
	}

	amountStr := fmt.Sprintf("%.2f", shortfall)
	subject := fmt.Sprintf("订阅套餐补差: %s", plan.DisplayName)
	var payURL string
	if req.ClientType == "mobile" {
		payURL, err = h.payment.CreateWapPay(order.OrderNo, amountStr, subject)
	} else {
		payURL, err = h.payment.CreatePagePay(order.OrderNo, amountStr, subject)
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create payment", "server_error", "payment_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"order_no":   order.OrderNo,
		"pay_url":    payURL,
		"expired_at": order.ExpiredAt,
	})
}


// HandleAdminListSubscriptions lists all subscriptions (admin).
// GET /api/admin/subscriptions
func (h *Handler) HandleAdminListSubscriptions(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	subs, total, err := h.store.ListUserSubscriptions(size, (page-1)*size)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list subscriptions", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"subscriptions": subs,
		"total":         total,
		"page":          page,
		"size":          size,
	})
}

type grantSubscriptionRequest struct {
	UserID string `json:"user_id"`
	PlanID int    `json:"plan_id"`
}

// HandleAdminGrantSubscription manually grants a subscription (admin).
// POST /api/admin/subscriptions/grant
func (h *Handler) HandleAdminGrantSubscription(w http.ResponseWriter, r *http.Request) {
	var req grantSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}
	if req.UserID == "" || req.PlanID <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "user_id and plan_id are required", "invalid_request", "missing_fields")
		return
	}

	sub, err := h.service.Subscribe(req.UserID, req.PlanID)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "subscription_error", "grant_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"subscription": sub})
}

// HandleAdminSubscriptionOrderStats returns aggregated subscription order statistics.
// GET /api/admin/subscription-orders/stats?days=30
func (h *Handler) HandleAdminSubscriptionOrderStats(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30
	}

	stats, err := h.store.GetSubscriptionOrderStats(days)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get stats", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// HandleAdminListSubscriptionOrders returns a paginated list of subscription orders.
// GET /api/admin/subscription-orders?page=1&size=20&status=paid&type=new
func (h *Handler) HandleAdminListSubscriptionOrders(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	status := r.URL.Query().Get("status")
	switch status {
	case "paid", "pending":
	default:
		status = ""
	}

	orderType := r.URL.Query().Get("type")
	switch orderType {
	case "new", "upgrade", "renew":
	default:
		orderType = ""
	}

	orders, total, err := h.store.ListAllSubscriptionOrders(size, (page-1)*size, status, orderType)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list subscription orders", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"orders": orders,
		"total":  total,
		"page":   page,
		"size":   size,
	})
}

// HandleAdminSubscriptionUsersUsage returns subscription users with their usage details.
// GET /api/admin/subscription-users-usage?page=1&size=20&search=&status=&plan_id=
func (h *Handler) HandleAdminSubscriptionUsersUsage(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	search := strings.TrimSpace(r.URL.Query().Get("search"))

	status := r.URL.Query().Get("status")
	switch status {
	case "active", "expired":
	default:
		status = ""
	}

	planFilter := r.URL.Query().Get("plan_id")

	users, total, activeCount, totalUsage, err := h.store.ListSubscriptionUsersUsage(size, (page-1)*size, search, status, planFilter)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list subscription users usage", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"users":        users,
		"total":        total,
		"active_count": activeCount,
		"total_usage":  totalUsage,
		"page":         page,
		"size":         size,
	})
}

// adminPlanRequest is the request body for creating/updating a subscription plan.
type adminPlanRequest struct {
	Name            string   `json:"name"`
	DisplayName     string   `json:"display_name"`
	Description     string   `json:"description"`
	MonthlyPriceCNY float64  `json:"monthly_price_cny"`
	QuotaAmountCNY  float64  `json:"quota_amount_cny"`
	DurationDays    int      `json:"duration_days"`
	SortOrder       int      `json:"sort_order"`
	Status          string   `json:"status"`
	Models          []string `json:"models"`
}

func (req *adminPlanRequest) validate() (string, string) {
	if strings.TrimSpace(req.Name) == "" {
		return "name is required", "missing_name"
	}
	if req.MonthlyPriceCNY < 0 {
		return "monthly_price_cny must be >= 0", "invalid_price"
	}
	if req.QuotaAmountCNY < 0 {
		return "quota_amount_cny must be >= 0", "invalid_quota"
	}
	if req.DurationDays <= 0 {
		return "duration_days must be > 0", "invalid_duration"
	}
	return "", ""
}

// HandleAdminListPlans lists all subscription plans regardless of status (admin).
// GET /api/admin/subscription-plans
func (h *Handler) HandleAdminListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.store.ListAllSubscriptionPlans()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list plans", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"plans": plans})
}

// HandleAdminCreatePlan creates a new subscription plan with model associations (admin).
// POST /api/admin/subscription-plans
func (h *Handler) HandleAdminCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req adminPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}
	if msg, code := req.validate(); msg != "" {
		httputil.WriteError(w, http.StatusBadRequest, msg, "invalid_request", code)
		return
	}

	plan, err := h.store.CreateSubscriptionPlan(store.SubscriptionPlan{
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		MonthlyPriceCNY: req.MonthlyPriceCNY,
		QuotaAmountCNY:  req.QuotaAmountCNY,
		DurationDays:    req.DurationDays,
		SortOrder:       req.SortOrder,
		Status:          req.Status,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create plan", "server_error", "db_error")
		return
	}

	if err := h.store.SetSubscriptionPlanModels(plan.ID, req.Models); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to set plan models", "server_error", "db_error")
		return
	}
	plan.Models = req.Models

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"plan": plan})
}

// HandleAdminUpdatePlan updates a plan's metadata and model associations (admin).
// PUT /api/admin/subscription-plans/{id}
func (h *Handler) HandleAdminUpdatePlan(w http.ResponseWriter, r *http.Request) {
	id, err := extractPlanID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid plan id", "invalid_request", "bad_id")
		return
	}

	var req adminPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}
	if msg, code := req.validate(); msg != "" {
		httputil.WriteError(w, http.StatusBadRequest, msg, "invalid_request", code)
		return
	}

	if err := h.store.UpdateSubscriptionPlan(store.SubscriptionPlan{
		ID:              id,
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		MonthlyPriceCNY: req.MonthlyPriceCNY,
		QuotaAmountCNY:  req.QuotaAmountCNY,
		DurationDays:    req.DurationDays,
		SortOrder:       req.SortOrder,
		Status:          req.Status,
	}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.WriteError(w, http.StatusNotFound, "plan not found", "invalid_request", "not_found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update plan", "server_error", "db_error")
		return
	}

	if err := h.store.SetSubscriptionPlanModels(id, req.Models); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to set plan models", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}

// HandleAdminDeletePlan removes a subscription plan (admin).
// DELETE /api/admin/subscription-plans/{id}
func (h *Handler) HandleAdminDeletePlan(w http.ResponseWriter, r *http.Request) {
	id, err := extractPlanID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid plan id", "invalid_request", "bad_id")
		return
	}
	if err := h.store.DeleteSubscriptionPlan(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.WriteError(w, http.StatusNotFound, "plan not found", "invalid_request", "not_found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete plan", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}

func extractPlanID(path string) (int, error) {
	path = strings.TrimRight(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return 0, fmt.Errorf("invalid path")
	}
	return strconv.Atoi(parts[len(parts)-1])
}
