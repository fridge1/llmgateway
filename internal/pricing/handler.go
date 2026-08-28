package pricing

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// PricingListResponse wraps model pricing rows (admin endpoint).
type PricingListResponse struct {
	Pricing []store.ModelPricing `json:"pricing"`
}

// PricingItem is a pricing row enriched with tenant discount metadata for the
// user-facing endpoint. The embedded ModelPricing carries the effective
// (possibly discounted) price under the existing field names; DiscountRate and
// OriginalPricing are only populated when the caller's tenant has a discount.
type PricingItem struct {
	store.ModelPricing
	DiscountRate    *float64            `json:"discount_rate,omitempty"`
	OriginalPricing *store.ModelPricing `json:"original_pricing,omitempty"`
}

// ActivePricingListResponse wraps enriched pricing rows (user endpoint).
type ActivePricingListResponse struct {
	Pricing []PricingItem `json:"pricing"`
}

// PricingHandler provides pricing API endpoints.
type PricingHandler struct {
	store store.Store
	cache *Cache
}

// NewPricingHandler creates a PricingHandler.
func NewPricingHandler(s store.Store, cache *Cache) *PricingHandler {
	return &PricingHandler{store: s, cache: cache}
}

func (h *PricingHandler) writePricingList(w http.ResponseWriter, prices []store.ModelPricing) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(PricingListResponse{
		Pricing: prices,
	})
}

// HandleListPricing returns all pricing records (admin only).
// GET /api/admin/pricing
func (h *PricingHandler) HandleListPricing(w http.ResponseWriter, r *http.Request) {
	prices, err := h.store.ListPricing()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list pricing", "server_error", "db_error")
		return
	}
	h.writePricingList(w, prices)
}

type upsertPricingRequest struct {
	InputPrice           float64                        `json:"input_price"`
	OutputPrice          float64                        `json:"output_price"`
	CachedInputPrice     float64                        `json:"cached_input_price"`
	CacheCreationPrice   float64                        `json:"cache_creation_price"`
	CacheCreation1hPrice float64                        `json:"cache_creation_1h_price"`
	BillingType          string                         `json:"billing_type"`
	IsActive             *bool                          `json:"is_active"`
	PricingTiers         []store.PricingTier            `json:"pricing_tiers,omitempty"`
	TimeBasedRules       []store.TimeBasedPricingRule   `json:"time_based_rules,omitempty"`
}

// HandleUpsertPricing creates or updates pricing for a model (admin only).
// PUT /api/admin/pricing/{model}
func (h *PricingHandler) HandleUpsertPricing(w http.ResponseWriter, r *http.Request) {
	const pricingPrefix = "/api/admin/pricing/"
	path := strings.TrimRight(r.URL.Path, "/")
	modelName := strings.TrimPrefix(path, pricingPrefix)
	if modelName == "" || modelName == path {
		httputil.WriteError(w, http.StatusBadRequest, "model name required", "invalid_request_error", "missing_model")
		return
	}
	// Decode URL-encoded model names (e.g. "pa%2Fclaude-sonnet-4-6" -> "pa/claude-sonnet-4-6")
	if decoded, err := url.PathUnescape(modelName); err == nil {
		modelName = decoded
	}

	var req upsertPricingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}

	if req.InputPrice < 0 || req.OutputPrice < 0 || req.CacheCreationPrice < 0 {
		httputil.WriteError(w, http.StatusBadRequest, "prices must be non-negative", "invalid_request_error", "invalid_price")
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// Capture old pricing for change log.
	oldPricing, _ := h.store.GetPricing(modelName)
	var oldValues map[string]any
	if oldPricing != nil {
		oldValues = map[string]any{
			"input_price":           oldPricing.InputPrice,
			"output_price":          oldPricing.OutputPrice,
			"cached_input_price":    oldPricing.CachedInputPrice,
			"cache_creation_price":  oldPricing.CacheCreationPrice,
			"cache_creation_1h_price": oldPricing.CacheCreation1hPrice,
			"billing_type":          oldPricing.BillingType,
			"is_active":             oldPricing.IsActive,
		}
	}

	if err := h.store.UpsertPricing(modelName, req.InputPrice, req.OutputPrice, req.CachedInputPrice, req.CacheCreationPrice, req.CacheCreation1hPrice, req.BillingType, isActive, req.PricingTiers, req.TimeBasedRules); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to upsert pricing", "server_error", "db_error")
		return
	}

	// Log the change.
	adminUserID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	newValues := map[string]any{
		"input_price":           req.InputPrice,
		"output_price":          req.OutputPrice,
		"cached_input_price":    req.CachedInputPrice,
		"cache_creation_price":  req.CacheCreationPrice,
		"cache_creation_1h_price": req.CacheCreation1hPrice,
		"billing_type":          req.BillingType,
		"is_active":             isActive,
	}
	if err := h.store.InsertPricingChangeLog(modelName, "pricing_update", adminUserID, oldValues, newValues); err != nil {
		slog.Warn("failed to log pricing change", "model", modelName, "error", err)
	}

	if h.cache != nil {
		h.cache.Invalidate(modelName)
		// Discount-based tenant prices derive from global pricing — drop them
		// so they recompute against the new global price.
		h.cache.InvalidateModelAllTenants(modelName)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleListActivePricing returns only active pricing records (all users).
// When the caller belongs to a tenant with custom pricing, model prices are
// returned as that tenant's effective (discounted) prices. For discount-based
// tenants, each item also carries the discount rate and the original global
// price so the UI can show a struck-through comparison.
// GET /api/pricing
func (h *PricingHandler) HandleListActivePricing(w http.ResponseWriter, r *http.Request) {
	prices, err := h.store.ListActivePricing()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list pricing", "server_error", "db_error")
		return
	}

	items := make([]PricingItem, len(prices))
	for i := range prices {
		items[i] = PricingItem{ModelPricing: prices[i]}
	}

	// Resolve the caller's pricing: prioritize user-specific pricing, then tenant pricing.
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID != "" && h.cache != nil {
		// First, check for user-specific pricing overrides.
		hasUserPricing := false
		for i := range items {
			eff, rate, original, used, gerr := h.cache.GetUserPricingDetail(userID, items[i].ModelName)
			if gerr != nil || eff == nil {
				continue
			}
			if used {
				hasUserPricing = true
				items[i].ModelPricing = *eff
				items[i].DiscountRate = rate
				if rate != nil && original != nil {
					items[i].OriginalPricing = original
				}
			}
		}

		// If no user pricing exists, fall back to tenant pricing.
		if !hasUserPricing {
			if tenantID, terr := h.store.GetUserPrimaryPricingTenant(userID); terr == nil && tenantID != "" {
				for i := range items {
					eff, rate, original, _, gerr := h.cache.GetTenantPricingDetail(tenantID, items[i].ModelName)
					if gerr != nil || eff == nil {
						continue
					}
					items[i].ModelPricing = *eff
					items[i].DiscountRate = rate
					if rate != nil && original != nil {
						items[i].OriginalPricing = original
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ActivePricingListResponse{Pricing: items})
}

// HandleListPricingChangeLogs returns paginated pricing change logs (admin only).
// GET /api/admin/pricing/change-logs
func (h *PricingHandler) HandleListPricingChangeLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	page := 1
	size := 20
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			size = n
		}
	}
	offset := (page - 1) * size

	logs, total, err := h.store.ListPricingChangeLogs(size, offset)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list change logs", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"logs":  logs,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
