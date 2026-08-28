package alerting

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler provides admin endpoints for alert rules and fired events.
type Handler struct {
	store store.Store
}

// NewHandler creates an alerting admin Handler.
func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

// HandleRules serves GET (list) and PUT (update one) on /api/admin/alert/rules.
func (h *Handler) HandleRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rules, err := h.store.ListAlertRules()
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to list alert rules", "server_error", "db_error")
			return
		}
		if rules == nil {
			rules = []store.AlertRule{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"rules": rules})
	case http.MethodPut:
		var req struct {
			ID              int   `json:"id"`
			Threshold       int64 `json:"threshold"`
			CooldownSeconds int   `json:"cooldown_seconds"`
			Enabled         bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
			return
		}
		if req.ID <= 0 || req.Threshold < 1 || req.CooldownSeconds < 0 {
			httputil.WriteError(w, http.StatusBadRequest, "invalid rule fields", "invalid_request_error", "bad_fields")
			return
		}
		if err := h.store.UpdateAlertRule(req.ID, req.Threshold, req.CooldownSeconds, req.Enabled); err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to update alert rule", "server_error", "db_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
	}
}

// HandleEvents returns the paginated fired-alert history.
// GET /api/admin/alert/events?page=1&size=20
func (h *Handler) HandleEvents(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	events, total, err := h.store.ListAlertEvents(size, (page-1)*size)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list alert events", "server_error", "db_error")
		return
	}
	if events == nil {
		events = []store.AlertEvent{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"events": events,
		"total":  total,
		"page":   page,
		"size":   size,
	})
}
