package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// ModelHandler provides CRUD endpoints for models.
type ModelHandler struct {
	store    store.Store
	onUpdate func() // called after model changes to rebuild router
}

// NewModelHandler creates a ModelHandler.
func NewModelHandler(s store.Store, onUpdate func()) *ModelHandler {
	return &ModelHandler{store: s, onUpdate: onUpdate}
}

type modelRequest struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Category    string            `json:"category"`
	Upstreams   []upstreamRequest `json:"upstreams"`
}

type upstreamRequest struct {
	Provider         string   `json:"provider"`
	Protocol         string   `json:"protocol"`
	Protocols        []string `json:"protocols"`
	UpstreamProvider string   `json:"upstream_provider"`
	UpstreamName     string   `json:"upstream_name"`
	BaseURL          string   `json:"base_url"`
	APIKey           string   `json:"api_key"`
	ModelOverride    string   `json:"model_override"`
	Weight           int      `json:"weight"`
}

// HandleList returns all models.
func (h *ModelHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	models, err := h.store.ListModels()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// HandleCreate adds a new model.
func (h *ModelHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req modelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}
	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "name is required", "invalid_request_error", "missing_name")
		return
	}

	upstreams := toStoreUpstreams(req.Upstreams)
	model, err := h.store.CreateModel(req.Name, req.DisplayName, req.Category, upstreams)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "create_failed")
		return
	}

	h.onUpdate()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(model)
}

// HandleUpdate modifies an existing model.
func (h *ModelHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid model ID", "invalid_request_error", "bad_id")
		return
	}

	var req modelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}

	upstreams := toStoreUpstreams(req.Upstreams)
	model, err := h.store.UpdateModel(id, req.Name, req.DisplayName, req.Category, upstreams)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "update_failed")
		return
	}

	h.onUpdate()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)
}

// HandleDelete removes a model.
func (h *ModelHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid model ID", "invalid_request_error", "bad_id")
		return
	}

	if err := h.store.DeleteModel(id); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "delete_failed")
		return
	}

	h.onUpdate()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func toStoreUpstreams(reqs []upstreamRequest) []store.Upstream {
	upstreams := make([]store.Upstream, len(reqs))
	for i, r := range reqs {
		w := r.Weight
		if w <= 0 {
			w = 1
		}
		upstreams[i] = store.Upstream{
			Provider:         r.Provider,
			Protocol:         r.Protocol,
			Protocols:        r.Protocols,
			UpstreamProvider: r.UpstreamProvider,
			UpstreamName:     r.UpstreamName,
			BaseURL:          r.BaseURL,
			APIKey:           r.APIKey,
			ModelOverride:    r.ModelOverride,
			Weight:           w,
		}
	}
	return upstreams
}

// extractID extracts the last path segment as an int64.
// e.g. /api/models/5 -> 5
func extractID(path string) (int64, error) {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	return strconv.ParseInt(parts[len(parts)-1], 10, 64)
}
