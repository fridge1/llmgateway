package admin

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// PricingInvalidator defines the interface for invalidating pricing cache entries.
type PricingInvalidator interface {
	InvalidateUser(userID, modelName string)
}

// UserPricingHandler handles user-specific pricing management endpoints.
type UserPricingHandler struct {
	store              store.Store
	pricingInvalidator PricingInvalidator
}

// NewUserPricingHandler creates a new UserPricingHandler.
func NewUserPricingHandler(s store.Store, invalidator PricingInvalidator) *UserPricingHandler {
	return &UserPricingHandler{
		store:              s,
		pricingInvalidator: invalidator,
	}
}

type upsertUserPricingRequest struct {
	InputPrice           float64             `json:"input_price"`
	OutputPrice          float64             `json:"output_price"`
	CachedInputPrice     float64             `json:"cached_input_price"`
	CacheCreationPrice   float64             `json:"cache_creation_price"`
	CacheCreation1hPrice float64             `json:"cache_creation_1h_price"`
	BillingType          string              `json:"billing_type"`
	IsActive             *bool               `json:"is_active"`
	PricingTiers         []store.PricingTier `json:"pricing_tiers,omitempty"`
	DiscountRate         *float64            `json:"discount_rate,omitempty"`
}

// HandleListUserPricing returns all pricing overrides for a user.
// GET /api/admin/users/{id}/pricing
func (h *UserPricingHandler) HandleListUserPricing(w http.ResponseWriter, r *http.Request) {
	userID := extractPathSegment(r.URL.Path, "users")
	prices, err := h.store.ListUserPricing(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list user pricing", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"pricing": prices,
	})
}

// HandleUpsertUserPricing creates or updates a user pricing override.
// PUT /api/admin/users/{id}/pricing/{model...}
func (h *UserPricingHandler) HandleUpsertUserPricing(w http.ResponseWriter, r *http.Request) {
	userID := extractPathSegment(r.URL.Path, "users")
	modelName := extractUserPricingModel(r.URL.Path)
	if modelName == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model name required", "invalid_request_error", "missing_model")
		return
	}

	var req upsertUserPricingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}
	if req.InputPrice < 0 || req.OutputPrice < 0 || req.CacheCreationPrice < 0 {
		httputil.WriteError(w, http.StatusBadRequest, "prices must be non-negative", "invalid_request_error", "invalid_price")
		return
	}
	if req.DiscountRate != nil && (*req.DiscountRate <= 0 || *req.DiscountRate > 10) {
		httputil.WriteError(w, http.StatusBadRequest, "pricing_factor must be in (0, 10]", "invalid_request_error", "invalid_pricing_factor")
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	adminUserID, _ := r.Context().Value(CtxUserIDKey).(string)

	if err := h.store.UpsertUserPricing(
		userID, modelName,
		req.InputPrice, req.OutputPrice, req.CachedInputPrice,
		req.CacheCreationPrice, req.CacheCreation1hPrice,
		req.BillingType, isActive, req.PricingTiers,
		req.DiscountRate, adminUserID,
	); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to upsert user pricing", "server_error", "db_error")
		return
	}

	if h.pricingInvalidator != nil {
		h.pricingInvalidator.InvalidateUser(userID, modelName)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleDeleteUserPricing removes a user pricing override.
// DELETE /api/admin/users/{id}/pricing/{model...}
func (h *UserPricingHandler) HandleDeleteUserPricing(w http.ResponseWriter, r *http.Request) {
	userID := extractPathSegment(r.URL.Path, "users")
	modelName := extractUserPricingModel(r.URL.Path)
	if modelName == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model name required", "invalid_request_error", "missing_model")
		return
	}

	if err := h.store.DeleteUserPricing(userID, modelName); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete user pricing", "server_error", "db_error")
		return
	}

	if h.pricingInvalidator != nil {
		h.pricingInvalidator.InvalidateUser(userID, modelName)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// extractUserPathModel extracts the model name following marker from a path
// like /api/admin/users/{id}/pricing/{model...}. Model names may contain
// slashes (e.g. "pa/claude-sonnet-4-6").
func extractUserPathModel(path, marker string) string {
	idx := strings.Index(path, marker)
	if idx < 0 {
		return ""
	}
	raw := strings.TrimSuffix(path[idx+len(marker):], "/")
	if decoded, err := url.PathUnescape(raw); err == nil {
		return decoded
	}
	return raw
}

// extractUserPricingModel extracts the model name from a path like
// /api/admin/users/{id}/pricing/{model...}
func extractUserPricingModel(path string) string {
	return extractUserPathModel(path, "/pricing/")
}
