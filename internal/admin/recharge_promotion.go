package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// RechargePromotionHandler manages recharge bonus campaigns.
type RechargePromotionHandler struct {
	store store.Store
}

// NewRechargePromotionHandler creates a RechargePromotionHandler.
func NewRechargePromotionHandler(s store.Store) *RechargePromotionHandler {
	return &RechargePromotionHandler{store: s}
}

type rechargePromotionRequest struct {
	Name              string    `json:"name"`
	StartsAt          time.Time `json:"starts_at"`
	EndsAt            time.Time `json:"ends_at"`
	BonusRatio        float64   `json:"bonus_ratio"`
	MinRechargeAmount float64   `json:"min_recharge_amount"`
	IsActive          *bool     `json:"is_active"`
}

func (req *rechargePromotionRequest) validate() (string, string) {
	if strings.TrimSpace(req.Name) == "" {
		return "活动名称不能为空", "missing_name"
	}
	if req.StartsAt.IsZero() || req.EndsAt.IsZero() {
		return "活动起止时间不能为空", "missing_time"
	}
	if !req.EndsAt.After(req.StartsAt) {
		return "结束时间必须晚于开始时间", "bad_time_range"
	}
	if req.BonusRatio <= 0 || req.BonusRatio > 1 {
		return "赠送比例必须在 (0, 1] 之间，例如 0.1 表示 10%", "bad_bonus_ratio"
	}
	if req.MinRechargeAmount < 0 {
		return "最低充值金额不能为负", "bad_min_amount"
	}
	return "", ""
}

func (req *rechargePromotionRequest) toModel() *store.RechargePromotion {
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	return &store.RechargePromotion{
		Name:              strings.TrimSpace(req.Name),
		StartsAt:          req.StartsAt,
		EndsAt:            req.EndsAt,
		BonusRatio:        req.BonusRatio,
		MinRechargeAmount: req.MinRechargeAmount,
		IsActive:          active,
	}
}

// HandleList returns all recharge promotions ordered by start time desc.
// GET /api/admin/recharge-promotions
func (h *RechargePromotionHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListRechargePromotions()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list recharge promotions", "server_error", "db_error")
		return
	}
	if list == nil {
		list = []store.RechargePromotion{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"promotions": list})
}

// HandleCreate creates a new recharge promotion.
// POST /api/admin/recharge-promotions
func (h *RechargePromotionHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req rechargePromotionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if msg, code := req.validate(); msg != "" {
		httputil.WriteError(w, http.StatusBadRequest, msg, "invalid_request_error", code)
		return
	}

	p, err := h.store.CreateRechargePromotion(req.toModel())
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create recharge promotion", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

// HandleUpdate updates an existing recharge promotion.
// PUT /api/admin/recharge-promotions/{id}
func (h *RechargePromotionHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := extractRechargePromotionID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid promotion id", "invalid_request_error", "bad_id")
		return
	}

	var req rechargePromotionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if msg, code := req.validate(); msg != "" {
		httputil.WriteError(w, http.StatusBadRequest, msg, "invalid_request_error", code)
		return
	}

	p, err := h.store.UpdateRechargePromotion(id, req.toModel())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.WriteError(w, http.StatusNotFound, "promotion not found", "invalid_request_error", "not_found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update recharge promotion", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// HandleDelete removes a recharge promotion.
// DELETE /api/admin/recharge-promotions/{id}
func (h *RechargePromotionHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := extractRechargePromotionID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid promotion id", "invalid_request_error", "bad_id")
		return
	}
	if err := h.store.DeleteRechargePromotion(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.WriteError(w, http.StatusNotFound, "promotion not found", "invalid_request_error", "not_found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete recharge promotion", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func extractRechargePromotionID(path string) (int, error) {
	path = strings.TrimRight(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return 0, fmt.Errorf("invalid path")
	}
	return strconv.Atoi(parts[len(parts)-1])
}
