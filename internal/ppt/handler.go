package ppt

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func getUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(admin.CtxUserIDKey).(string)
	return userID, ok && userID != ""
}

// SubmitTask handles POST /api/ppt/tasks
func (h *Handler) SubmitTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	var req struct {
		Model          string `json:"model"`
		Topic          string `json:"topic"`
		SlideCount     int    `json:"slide_count"`
		Language       string `json:"language"`
		Theme          string `json:"theme"`
		Audience       string `json:"audience"`
		Tone           string `json:"tone"`
		Purpose        string `json:"purpose"`
		OutlineOnly    bool   `json:"outline_only"`
		GenerateImages *bool  `json:"generate_images"`
		ContextText    string `json:"context_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalid_request_error", "invalid_json")
		return
	}

	if req.Model == "" || req.Topic == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model and topic are required", "invalid_request_error", "missing_fields")
		return
	}
	if req.SlideCount < 4 || req.SlideCount > 30 {
		req.SlideCount = 8
	}
	if req.Language == "" {
		req.Language = "zh"
	}
	if req.Theme == "" {
		req.Theme = "business-blue"
	}
	if req.Audience == "" {
		req.Audience = "general"
	}
	if req.Tone == "" {
		req.Tone = "professional"
	}
	if req.Purpose == "" {
		req.Purpose = "inform"
	}
	if len(req.ContextText) > 50000 {
		httputil.WriteError(w, http.StatusBadRequest, "context_text exceeds 50KB limit", "invalid_request_error", "context_too_large")
		return
	}
	generateImages := true
	if req.GenerateImages != nil {
		generateImages = *req.GenerateImages
	}

	task, err := h.service.SubmitTask(r.Context(), userID, req.Model, req.Topic, req.SlideCount, req.Language, req.Theme, req.Audience, req.Tone, req.Purpose, req.OutlineOnly, generateImages, req.ContextText)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient balance") {
			httputil.WriteError(w, http.StatusPaymentRequired, "余额不足", "insufficient_balance", "insufficient_balance")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to submit task", "server_error", "submit_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     task.ID,
		"status": task.Status,
	})
}

// ListTasks handles GET /api/ppt/tasks
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	tasks, err := h.service.store.GetPptTasks(r.Context(), userID, limit, offset)
	if err != nil {
		slog.Error("GetPptTasks failed", "error", err, "user_id", userID, "limit", limit, "offset", offset)
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to list tasks", "server_error", "list_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GetTask handles GET /api/ppt/tasks/{id}
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		httputil.WriteError(w, http.StatusBadRequest, "missing task ID", "invalid_request_error", "missing_id")
		return
	}
	idStr := parts[len(parts)-1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid task ID", "invalid_request_error", "invalid_id")
		return
	}

	task, err := h.service.store.GetPptTask(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "Task not found", "not_found", "task_not_found")
		return
	}

	if task.UserID != userID {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "auth_error", "not_owner")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// ConfirmOutline handles POST /api/ppt/tasks/{id}/confirm
func (h *Handler) ConfirmOutline(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	// Extract task ID from path: /api/ppt/tasks/{id}/confirm
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		httputil.WriteError(w, http.StatusBadRequest, "missing task ID", "invalid_request_error", "missing_id")
		return
	}
	// ID is second to last (before "confirm")
	idStr := parts[len(parts)-2]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid task ID", "invalid_request_error", "invalid_id")
		return
	}

	if err := h.service.ConfirmOutline(r.Context(), id, userID); err != nil {
		if strings.Contains(err.Error(), "forbidden") {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "auth_error", "not_owner")
			return
		}
		if strings.Contains(err.Error(), "not in outline_ready") {
			httputil.WriteError(w, http.StatusConflict, "Task is not awaiting outline confirmation", "invalid_state", "not_outline_ready")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to confirm outline", "server_error", "confirm_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "confirmed",
	})
}

// UpdatePresentation handles PUT /api/ppt/tasks/{id}/presentation
func (h *Handler) UpdatePresentation(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		httputil.WriteError(w, http.StatusBadRequest, "missing task ID", "invalid_request_error", "missing_id")
		return
	}
	idStr := parts[len(parts)-2]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid task ID", "invalid_request_error", "invalid_id")
		return
	}

	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalid_request_error", "invalid_json")
		return
	}

	if err := h.service.store.UpdatePptPresentation(r.Context(), id, userID, body); err != nil {
		if strings.Contains(err.Error(), "not found or not completed") {
			httputil.WriteError(w, http.StatusConflict, "Task not found or not in completed status", "invalid_state", "not_completed")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to save presentation", "server_error", "save_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "saved",
	})
}

// DeleteTask handles DELETE /api/ppt/tasks/{id}
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		httputil.WriteError(w, http.StatusBadRequest, "missing task ID", "invalid_request_error", "missing_id")
		return
	}
	idStr := parts[len(parts)-1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid task ID", "invalid_request_error", "invalid_id")
		return
	}

	if err := h.service.store.DeletePptTask(r.Context(), id, userID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.WriteError(w, http.StatusNotFound, "Task not found", "not_found", "task_not_found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to delete task", "server_error", "delete_failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
