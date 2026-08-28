package notifier

import (
	"encoding/json"
	"net/http"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler serves the user notification-preference endpoints.
type Handler struct {
	store store.Store
}

// NewHandler creates a notifier preference Handler.
func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

// HandlePreferences serves GET/PUT /api/notification/preferences.
// GET returns every configurable event type with its SMS switch (default off).
func (h *Handler) HandlePreferences(w http.ResponseWriter, r *http.Request) {
	uid, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if uid == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "unauthorized")
		return
	}
	switch r.Method {
	case http.MethodGet:
		stored, err := h.store.ListNotificationPreferences(uid)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to list preferences", "server_error", "db_error")
			return
		}
		smsOn := map[string]bool{}
		for _, p := range stored {
			if p.Channel == "sms" {
				smsOn[p.EventType] = p.Enabled
			}
		}
		type pref struct {
			EventType string `json:"event_type"`
			SMS       bool   `json:"sms"`
		}
		out := make([]pref, 0, len(store.NotificationEventTypes))
		for _, e := range store.NotificationEventTypes {
			out = append(out, pref{EventType: e, SMS: smsOn[e]})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"preferences": out})
	case http.MethodPut:
		var req struct {
			EventType string `json:"event_type"`
			SMS       bool   `json:"sms"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
			return
		}
		if !store.ValidNotificationEventType(req.EventType) {
			httputil.WriteError(w, http.StatusBadRequest, "unknown event type", "invalid_request_error", "bad_event_type")
			return
		}
		if err := h.store.UpsertNotificationPreference(uid, req.EventType, "sms", req.SMS); err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to update preference", "server_error", "db_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
	}
}
