package rechargelottery

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler provides recharge lottery API endpoints.
type Handler struct {
	store store.Store
}

// NewHandler creates a new recharge lottery Handler.
func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

// HandleAdminGet handles GET /api/admin/recharge-lottery — returns active config.
func (h *Handler) HandleAdminGet(w http.ResponseWriter, r *http.Request) {
	lottery, err := h.store.GetActiveLottery()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get lottery", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"lottery": lottery})
}

type createRequest struct {
	Name         string `json:"name"`
	TriggerEvery int    `json:"trigger_every"`
}

// HandleAdminCreate handles POST /api/admin/recharge-lottery — creates a new lottery.
func (h *Handler) HandleAdminCreate(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.TriggerEvery <= 0 {
		req.TriggerEvery = 10
	}
	if req.Name == "" {
		req.Name = "充值幸运奖"
	}
	lottery, err := h.store.CreateRechargeLottery(req.Name, req.TriggerEvery)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create lottery", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(lottery)
}

type updateRequest struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	TriggerEvery int    `json:"trigger_every"`
}

// HandleAdminUpdate handles PUT /api/admin/recharge-lottery/{id} — updates lottery config.
func (h *Handler) HandleAdminUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/recharge-lottery/")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid id", "invalid_request_error", "bad_id")
		return
	}
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.Status != "active" && req.Status != "paused" {
		httputil.WriteError(w, http.StatusBadRequest, "status must be active or paused", "invalid_request_error", "bad_status")
		return
	}
	if req.TriggerEvery <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "trigger_every must be positive", "invalid_request_error", "bad_trigger")
		return
	}
	lottery, err := h.store.UpdateRechargeLottery(id, req.Name, req.Status, req.TriggerEvery)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update lottery", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lottery)
}

// HandleAdminRounds handles GET /api/admin/recharge-lottery/{id}/rounds — paginated draw history.
func (h *Handler) HandleAdminRounds(w http.ResponseWriter, r *http.Request) {
	// Path: /api/admin/recharge-lottery/{id}/rounds
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/recharge-lottery/")
	idStr := strings.TrimSuffix(path, "/rounds")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid id", "invalid_request_error", "bad_id")
		return
	}
	page, size := 1, 20
	if v := r.URL.Query().Get("page"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("size"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 && n <= 100 {
			size = n
		}
	}
	rounds, total, err := h.store.ListRechargeLotteryRoundsAdmin(id, size, (page-1)*size)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list rounds", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"rounds": rounds, "total": total, "page": page, "size": size})
}

// HandlePublicGet handles GET /api/recharge-lottery — returns active lottery config with progress (public).
func (h *Handler) HandlePublicGet(w http.ResponseWriter, r *http.Request) {
	lottery, err := h.store.GetActiveLottery()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get lottery", "server_error", "db_error")
		return
	}
	var currentEntries int
	if lottery != nil {
		currentEntries, _ = h.store.GetCurrentRoundEntryCount(lottery.ID)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"lottery": lottery, "current_entries": currentEntries})
}

// HandlePublicRounds handles GET /api/recharge-lottery/rounds — recent winners (public).
func (h *Handler) HandlePublicRounds(w http.ResponseWriter, r *http.Request) {
	lottery, err := h.store.GetActiveLottery()
	if err != nil || lottery == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"rounds": []any{}})
		return
	}
	rounds, _, err := h.store.ListRechargeLotteryRounds(lottery.ID, 20, 0)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list rounds", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"rounds": rounds})
}
