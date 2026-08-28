// Package referral serves the invite-code API: each user can view their code,
// invite link, and referral performance.
package referral

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler serves referral endpoints.
type Handler struct {
	store store.Store
}

// NewHandler creates a referral handler.
func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

// HandleInfo returns the authenticated user's invite code and referral stats.
func (h *Handler) HandleInfo(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	info, err := h.store.GetReferralInfo(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "referral_info")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

// ---------------------------------------------------------------------------
// Admin: referral rule configuration
// ---------------------------------------------------------------------------

// HandleAdminRules serves GET (list) / POST (append new rule) on
// /api/admin/referral/rules. Rules are append-only; the newest enabled,
// effective rule wins, so a "change" is a new row (auditable history).
func (h *Handler) HandleAdminRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rules, err := h.store.ListReferralRules(50)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to list referral rules", "server_error", "db_error")
			return
		}
		if rules == nil {
			rules = []store.ReferralRule{}
		}
		active, _ := h.store.GetActiveReferralRule()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"rules": rules, "active": active})
	case http.MethodPost:
		adminID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
		var req struct {
			InviterBonusCNY     float64 `json:"inviter_bonus_cny"`
			InviteeBonusCNY     float64 `json:"invitee_bonus_cny"`
			MinFirstRechargeCNY float64 `json:"min_first_recharge_cny"`
			Enabled             bool    `json:"enabled"`
			EffectiveFrom       string  `json:"effective_from"` // RFC3339 or empty = now
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
			return
		}
		if req.InviterBonusCNY < 0 || req.InviteeBonusCNY < 0 || req.MinFirstRechargeCNY < 0 ||
			req.InviterBonusCNY > 1000 || req.InviteeBonusCNY > 1000 {
			httputil.WriteError(w, http.StatusBadRequest, "奖励金额需在 0-1000 之间", "invalid_request_error", "bad_amount")
			return
		}
		effective := time.Now()
		if req.EffectiveFrom != "" {
			t, err := time.Parse(time.RFC3339, req.EffectiveFrom)
			if err != nil {
				httputil.WriteError(w, http.StatusBadRequest, "effective_from 需为 RFC3339 格式", "invalid_request_error", "bad_time")
				return
			}
			effective = t
		}
		rule, err := h.store.CreateReferralRule(req.InviterBonusCNY, req.InviteeBonusCNY, req.MinFirstRechargeCNY, req.Enabled, effective, adminID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to create referral rule", "server_error", "db_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(rule)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
	}
}
