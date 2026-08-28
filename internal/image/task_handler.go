package image

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/imageshare"
	"github.com/zhulang/llm-gateway/internal/store"
)

// TaskHandler handles async image task API requests.
type TaskHandler struct {
	service    *Service
	shareStore *imageshare.Store
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(service *Service) *TaskHandler {
	return &TaskHandler{service: service}
}

// SetImageShareStore wires the image-share store for quota validation. Pass nil to disable.
func (h *TaskHandler) SetImageShareStore(s *imageshare.Store) {
	h.shareStore = s
}

// validateAndPrepareShare returns (shareKeyID, ok). When isShare is false, returns ("", true).
// On error it writes the response and returns ("", false).
//
// For image-share requests, it atomically reserves n quota slots up front via
// IncrementUsed. This ensures concurrent submissions can't all pass an
// optimistic "remaining >= n" check and then all hit the upstream provider.
// Callers must invoke refundShareQuota when the downstream service fails to
// enqueue the task, otherwise the reservation leaks.
func (h *TaskHandler) validateAndPrepareShare(w http.ResponseWriter, r *http.Request, n int) (string, bool) {
	if !imageshare.IsImageShareRequest(r) {
		return "", true
	}
	if h.shareStore == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "image-share not configured", "server_error", "share_unavailable")
		return "", false
	}
	keyID := imageshare.KeyIDFromContext(r.Context())
	if keyID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "missing share context", "auth_error", "missing_share_ctx")
		return "", false
	}
	if _, err := h.shareStore.IncrementUsed(keyID, n); err != nil {
		if errors.Is(err, imageshare.ErrQuotaExhausted) {
			// Either quota is exhausted or the key is no longer active. The
			// IncrementUsed WHERE clause covers both, and exposing the same
			// 4xx avoids leaking whether the key was disabled vs out of quota.
			httputil.WriteError(w, http.StatusPaymentRequired, "图片生成次数不足，请联系管理员充值或更换密钥", "insufficient_quota", "quota_insufficient")
			return "", false
		}
		httputil.WriteError(w, http.StatusInternalServerError, "图片配额预占失败，请稍后重试", "server_error", "quota_reserve_error")
		return "", false
	}
	return keyID, true
}

// refundShareQuota releases n previously reserved quota slots. Safe to call
// with an empty key id; logs and swallows DB errors since the caller is
// already in an error path.
func (h *TaskHandler) refundShareQuota(keyID string, n int) {
	if keyID == "" || h.shareStore == nil || n <= 0 {
		return
	}
	if err := h.shareStore.RefundUsed(keyID, n); err != nil {
		slog.Warn("imageshare refund after submit failure", "key", keyID, "n", n, "error", err)
	}
}

// SubmitGenerate handles POST /api/image/tasks
func (h *TaskHandler) SubmitGenerate(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	var req struct {
		Model  string         `json:"model"`
		Prompt string         `json:"prompt"`
		Size   string         `json:"size"`
		N      int            `json:"n"`
		Params map[string]any `json:"params,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalid_request_error", "invalid_json")
		return
	}
	if req.Model == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
		return
	}
	if req.Prompt == "" {
		httputil.WriteError(w, http.StatusBadRequest, "prompt is required", "invalid_request_error", "missing_prompt")
		return
	}
	if utf8.RuneCountInString(req.Prompt) > 2000 {
		httputil.WriteError(w, http.StatusBadRequest, "prompt too long (max 2000 characters)", "invalid_request_error", "prompt_too_long")
		return
	}
	if req.Size == "" {
		req.Size = "1024x1024"
	}
	if req.N < 1 || req.N > 4 {
		req.N = 1
	}

	if imageshare.IsImageShareRequest(r) && req.Model != imageshare.AllowedModel {
		httputil.WriteError(w, http.StatusForbidden, "图片分发仅支持 gpt-image-2 模型", "auth_error", "model_not_allowed")
		return
	}

	shareKeyID, ok := h.validateAndPrepareShare(w, r, req.N)
	if !ok {
		return
	}

	task, err := h.service.SubmitGenerateTask(r.Context(), userID, req.Model, req.Prompt, req.Size, req.N, req.Params, shareKeyID)
	if err != nil {
		h.refundShareQuota(shareKeyID, req.N)
		if strings.Contains(err.Error(), "service unavailable") || strings.Contains(err.Error(), "storage not configured") {
			httputil.WriteError(w, http.StatusServiceUnavailable, "图片生成服务暂时不可用", "service_unavailable", "tos_not_configured")
			return
		}
		if strings.Contains(err.Error(), "insufficient balance") {
			httputil.WriteError(w, http.StatusPaymentRequired, "余额不足，请充值", "insufficient_balance", "insufficient_balance")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, extractUpstreamMessage(err), "server_error", "submit_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     task.ID,
		"status": task.Status,
	})
}

// SubmitEdit handles POST /api/image/tasks/edit (multipart/form-data)
func (h *TaskHandler) SubmitEdit(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid multipart form", "invalid_request_error", "invalid_form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	model := r.FormValue("model")
	prompt := r.FormValue("prompt")
	size := r.FormValue("size")
	n, _ := strconv.Atoi(r.FormValue("n"))

	var params map[string]any
	if raw := r.FormValue("params"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &params); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid params json", "invalid_request_error", "invalid_params")
			return
		}
	}

	if model == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
		return
	}
	if prompt == "" {
		httputil.WriteError(w, http.StatusBadRequest, "prompt is required", "invalid_request_error", "missing_prompt")
		return
	}
	if utf8.RuneCountInString(prompt) > 32000 {
		httputil.WriteError(w, http.StatusBadRequest, "prompt too long", "invalid_request_error", "prompt_too_long")
		return
	}
	if size == "" {
		size = "1024x1024"
	}
	if n < 1 || n > 4 {
		n = 1
	}

	if imageshare.IsImageShareRequest(r) && model != imageshare.AllowedModel {
		httputil.WriteError(w, http.StatusForbidden, "图片分发仅支持 gpt-image-2 模型", "auth_error", "model_not_allowed")
		return
	}

	shareKeyID, ok := h.validateAndPrepareShare(w, r, n)
	if !ok {
		return
	}

	imageFiles := r.MultipartForm.File["image[]"]
	if len(imageFiles) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "at least one image is required", "invalid_request_error", "missing_image")
		return
	}
	if len(imageFiles) > 4 {
		httputil.WriteError(w, http.StatusBadRequest, "maximum 4 images allowed", "invalid_request_error", "too_many_images")
		return
	}

	var images [][]byte
	for _, fh := range imageFiles {
		f, err := fh.Open()
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, fmt.Sprintf("failed to read image: %v", err), "invalid_request_error", "bad_image")
			return
		}
		data, err := io.ReadAll(io.LimitReader(f, 25<<20))
		f.Close()
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "failed to read image data", "invalid_request_error", "bad_image")
			return
		}
		images = append(images, data)
	}

	var maskData []byte
	if maskFiles := r.MultipartForm.File["mask"]; len(maskFiles) > 0 {
		f, err := maskFiles[0].Open()
		if err == nil {
			maskData, _ = io.ReadAll(io.LimitReader(f, 25<<20))
			f.Close()
		}
	}

	task, err := h.service.SubmitEditTask(r.Context(), userID, model, prompt, size, n, images, maskData, params, shareKeyID)
	if err != nil {
		h.refundShareQuota(shareKeyID, n)
		if strings.Contains(err.Error(), "service unavailable") || strings.Contains(err.Error(), "storage not configured") {
			httputil.WriteError(w, http.StatusServiceUnavailable, "图片编辑服务暂时不可用", "service_unavailable", "tos_not_configured")
			return
		}
		if strings.Contains(err.Error(), "insufficient balance") {
			httputil.WriteError(w, http.StatusPaymentRequired, "余额不足，请充值", "insufficient_balance", "insufficient_balance")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, extractUpstreamMessage(err), "server_error", "submit_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     task.ID,
		"status": task.Status,
	})
}

// ListTasks handles GET /api/image/tasks
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			limit = l
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			offset = o
		}
	}

	var (
		tasks []store.ImageTask
		err   error
	)
	if imageshare.IsImageShareRequest(r) {
		keyID := imageshare.KeyIDFromContext(r.Context())
		if keyID == "" {
			httputil.WriteError(w, http.StatusUnauthorized, "missing share context", "auth_error", "missing_share_ctx")
			return
		}
		tasks, err = h.service.store.GetImageTasksByShareKey(r.Context(), keyID, limit, offset)
	} else {
		tasks, err = h.service.store.GetImageTasks(r.Context(), userID, limit, offset)
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to get tasks", "server_error", "query_failed")
		return
	}
	if tasks == nil {
		tasks = []store.ImageTask{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GetTask handles GET /api/image/tasks/:id
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid task ID", "invalid_request_error", "invalid_id")
		return
	}
	taskID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid task ID", "invalid_request_error", "invalid_id")
		return
	}

	task, err := h.service.store.GetImageTask(r.Context(), taskID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "Task not found", "invalid_request_error", "task_not_found")
		return
	}
	if imageshare.IsImageShareRequest(r) {
		keyID := imageshare.KeyIDFromContext(r.Context())
		if task.ImageShareKeyID == nil || *task.ImageShareKeyID != keyID {
			httputil.WriteError(w, http.StatusForbidden, "not authorized", "auth_error", "not_authorized")
			return
		}
	} else {
		if task.UserID != userID || task.ImageShareKeyID != nil {
			httputil.WriteError(w, http.StatusForbidden, "not authorized", "auth_error", "not_authorized")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// extractTaskID parses /api/image/tasks/{id} or /api/image/tasks/{id}/images and returns id.
func extractTaskID(path string) (int, error) {
	trimmed := strings.TrimSuffix(path, "/")
	if strings.HasSuffix(trimmed, "/images") {
		trimmed = strings.TrimSuffix(trimmed, "/images")
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid path")
	}
	return strconv.Atoi(parts[len(parts)-1])
}

// DeleteTask handles DELETE /api/image/tasks/:id
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	taskID, err := extractTaskID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid task ID", "invalid_request_error", "invalid_id")
		return
	}
	if imageshare.IsImageShareRequest(r) {
		keyID := imageshare.KeyIDFromContext(r.Context())
		if keyID == "" {
			httputil.WriteError(w, http.StatusUnauthorized, "missing share context", "auth_error", "missing_share_ctx")
			return
		}
		if err := h.service.DeleteTaskAsImageShare(r.Context(), keyID, taskID); err != nil {
			writeDeleteError(w, err)
			return
		}
	} else {
		if err := h.service.DeleteTask(r.Context(), userID, taskID); err != nil {
			writeDeleteError(w, err)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeleteResultImage handles DELETE /api/image/tasks/:id/images?url=...
func (h *TaskHandler) DeleteResultImage(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	taskID, err := extractTaskID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid task ID", "invalid_request_error", "invalid_id")
		return
	}
	url := r.URL.Query().Get("url")
	if url == "" {
		httputil.WriteError(w, http.StatusBadRequest, "url query parameter is required", "invalid_request_error", "missing_url")
		return
	}
	var remaining int
	if imageshare.IsImageShareRequest(r) {
		keyID := imageshare.KeyIDFromContext(r.Context())
		if keyID == "" {
			httputil.WriteError(w, http.StatusUnauthorized, "missing share context", "auth_error", "missing_share_ctx")
			return
		}
		remaining, err = h.service.DeleteResultImageAsImageShare(r.Context(), keyID, taskID, url)
	} else {
		remaining, err = h.service.DeleteResultImage(r.Context(), userID, taskID, url)
	}
	if err != nil {
		writeDeleteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"remaining": remaining,
	})
}

// writeDeleteError maps service errors to HTTP responses.
func writeDeleteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrTaskNotFound):
		httputil.WriteError(w, http.StatusNotFound, "Task not found", "invalid_request_error", "task_not_found")
	case errors.Is(err, ErrTaskForbidden):
		httputil.WriteError(w, http.StatusForbidden, "not authorized", "auth_error", "not_authorized")
	case errors.Is(err, ErrImageNotInTask):
		httputil.WriteError(w, http.StatusNotFound, "image not found in task", "invalid_request_error", "image_not_found")
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to delete", "server_error", "delete_failed")
	}
}
