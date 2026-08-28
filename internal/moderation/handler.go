package moderation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler provides admin endpoints for moderation settings, keywords and hits.
type Handler struct {
	store store.Store
	svc   *Service
}

// NewHandler creates a moderation admin Handler. svc is refreshed after writes.
func NewHandler(s store.Store, svc *Service) *Handler {
	return &Handler{store: s, svc: svc}
}

// HandleSettings serves GET/PUT /api/admin/moderation/settings.
func (h *Handler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := h.store.GetModerationSettings()
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to get moderation settings", "server_error", "db_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	case http.MethodPut:
		var req struct {
			Enabled    bool `json:"enabled"`
			EnforceAll bool `json:"enforce_all"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
			return
		}
		if err := h.store.UpdateModerationSettings(req.Enabled, req.EnforceAll); err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to update moderation settings", "server_error", "db_error")
			return
		}
		h.svc.Refresh()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
	}
}

// HandleKeywords serves GET (list) / POST (create) /api/admin/moderation/keywords.
func (h *Handler) HandleKeywords(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		kws, err := h.store.ListModerationKeywords()
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to list keywords", "server_error", "db_error")
			return
		}
		if kws == nil {
			kws = []store.ModerationKeyword{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"keywords": kws})
	case http.MethodPost:
		var req struct {
			Keyword  string `json:"keyword"`
			Category string `json:"category"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
			return
		}
		req.Keyword = strings.TrimSpace(req.Keyword)
		if req.Keyword == "" || len(req.Keyword) > 128 {
			httputil.WriteError(w, http.StatusBadRequest, "keyword is required (max 128 chars)", "invalid_request_error", "bad_keyword")
			return
		}
		if req.Category == "" {
			req.Category = "general"
		}
		k, err := h.store.CreateModerationKeyword(req.Keyword, req.Category)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to create keyword", "server_error", "db_error")
			return
		}
		h.svc.Refresh()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(k)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
	}
}

// HandleDeleteKeyword serves DELETE /api/admin/moderation/keywords/{id}.
func (h *Handler) HandleDeleteKeyword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
	id, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil || id <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid keyword id", "invalid_request_error", "bad_id")
		return
	}
	if err := h.store.DeleteModerationKeyword(id); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "keyword not found", "invalid_request_error", "not_found")
		return
	}
	h.svc.Refresh()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleHits serves GET /api/admin/moderation/hits?page=&size=&user_id=&from=&to=.
func (h *Handler) HandleHits(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	var from, to *time.Time
	if s := r.URL.Query().Get("from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			from = &t
		}
	}
	if s := r.URL.Query().Get("to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			end := t.Add(24 * time.Hour)
			to = &end
		}
	}
	hits, total, err := h.store.ListModerationHits(r.URL.Query().Get("user_id"), from, to, size, (page-1)*size)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list moderation hits", "server_error", "db_error")
		return
	}
	if hits == nil {
		hits = []store.ModerationHit{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"hits":  hits,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
