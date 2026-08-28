// Package checkin serves the daily check-in API: a streak-based engagement
// mechanic that credits a small escalating reward to the user's balance.
package checkin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler serves check-in endpoints.
type Handler struct {
	store     store.Store
	cfgHolder *config.Holder
}

// NewHandler creates a check-in handler.
func NewHandler(s store.Store, cfgHolder *config.Holder) *Handler {
	return &Handler{store: s, cfgHolder: cfgHolder}
}

func (h *Handler) base() float64 {
	if h.cfgHolder == nil {
		return 0
	}
	return h.cfgHolder.Get().Retention.CheckinBaseRewardCNY
}

// HandleStatus returns the current check-in state for the authenticated user.
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	status, err := h.store.GetCheckinStatus(userID, h.base())
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "checkin_status")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// HandleCheckin performs today's check-in for the authenticated user.
func (h *Handler) HandleCheckin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method")
		return
	}
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	result, err := h.store.DoCheckin(userID, h.base())
	if errors.Is(err, store.ErrAlreadyCheckedIn) {
		httputil.WriteError(w, http.StatusConflict, "今日已签到", "invalid_request_error", "already_checked_in")
		return
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "checkin")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
