package image

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/proxy"
	"github.com/zhulang/llm-gateway/internal/store"
)

// PublicTaskHandler exposes the async image-task pipeline to API-key callers
// (user / tenant / sub-user keys) under /v1/images/tasks. It is independent of
// the JWT/image-share TaskHandler and shares only the underlying Service.
type PublicTaskHandler struct {
	service *Service
}

// NewPublicTaskHandler creates a new PublicTaskHandler.
func NewPublicTaskHandler(service *Service) *PublicTaskHandler {
	return &PublicTaskHandler{service: service}
}

const (
	publicTaskMaxImages    = 4
	publicTaskMaxImageByte = 25 << 20 // 25MB per decoded input image
)

// writeSubmitError maps service submit errors to HTTP responses.
func writeSubmitError(w http.ResponseWriter, err error) {
	switch {
	case strings.Contains(err.Error(), "service unavailable") || strings.Contains(err.Error(), "storage not configured"):
		httputil.WriteError(w, http.StatusServiceUnavailable, "图片服务暂时不可用", "service_unavailable", "tos_not_configured")
	case strings.Contains(err.Error(), "insufficient balance"):
		httputil.WriteError(w, http.StatusPaymentRequired, "余额不足或配额不足，请充值", "insufficient_balance", "insufficient_balance")
	default:
		httputil.WriteError(w, http.StatusInternalServerError, extractUpstreamMessage(err), "server_error", "submit_failed")
	}
}

// SubmitGenerate handles POST /v1/images/tasks.
func (h *PublicTaskHandler) SubmitGenerate(w http.ResponseWriter, r *http.Request) {
	auth, err := h.service.core.AuthenticateAny(r)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
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
	if req.N < 1 || req.N > publicTaskMaxImages {
		req.N = 1
	}
	if err := h.service.core.CheckModelAccess(auth, req.Model); err != nil {
		httputil.WriteError(w, http.StatusForbidden, err.Error(), "auth_error", "model_not_allowed")
		return
	}

	task, err := h.service.SubmitGeneratePublic(r.Context(), auth, req.Model, req.Prompt, req.Size, req.N, req.Params)
	if err != nil {
		writeSubmitError(w, err)
		return
	}
	writeTaskAccepted(w, task.ID, task.Status)
}

// SubmitEdit handles POST /v1/images/tasks/edits (JSON + base64/URL inputs).
func (h *PublicTaskHandler) SubmitEdit(w http.ResponseWriter, r *http.Request) {
	auth, err := h.service.core.AuthenticateAny(r)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
		return
	}

	var req struct {
		Model        string         `json:"model"`
		Prompt       string         `json:"prompt"`
		Size         string         `json:"size"`
		N            int            `json:"n"`
		ImageURLs    []string       `json:"image_urls,omitempty"`
		ImageBase64s []string       `json:"image_base64s,omitempty"`
		MaskBase64   string         `json:"mask_base64,omitempty"`
		Params       map[string]any `json:"params,omitempty"`
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
	if utf8.RuneCountInString(req.Prompt) > 32000 {
		httputil.WriteError(w, http.StatusBadRequest, "prompt too long", "invalid_request_error", "prompt_too_long")
		return
	}
	if req.Size == "" {
		req.Size = "1024x1024"
	}
	if req.N < 1 || req.N > publicTaskMaxImages {
		req.N = 1
	}
	if err := h.service.core.CheckModelAccess(auth, req.Model); err != nil {
		httputil.WriteError(w, http.StatusForbidden, err.Error(), "auth_error", "model_not_allowed")
		return
	}

	images, ok := h.collectInputImages(w, r, req.ImageBase64s, req.ImageURLs)
	if !ok {
		return
	}

	var mask []byte
	if req.MaskBase64 != "" {
		mask, err = base64.StdEncoding.DecodeString(req.MaskBase64)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid mask_base64", "invalid_request_error", "invalid_mask")
			return
		}
	}

	task, err := h.service.SubmitEditPublic(r.Context(), auth, req.Model, req.Prompt, req.Size, req.N, images, mask, req.Params)
	if err != nil {
		writeSubmitError(w, err)
		return
	}
	writeTaskAccepted(w, task.ID, task.Status)
}

// collectInputImages decodes base64 images and downloads URL images, enforcing
// the count/size limits. On error it writes the response and returns ok=false.
func (h *PublicTaskHandler) collectInputImages(w http.ResponseWriter, r *http.Request, b64s, urls []string) ([][]byte, bool) {
	var images [][]byte
	for _, b64 := range b64s {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid image_base64s entry", "invalid_request_error", "bad_image")
			return nil, false
		}
		if len(data) > publicTaskMaxImageByte {
			httputil.WriteError(w, http.StatusBadRequest, "input image too large (max 25MB)", "invalid_request_error", "image_too_large")
			return nil, false
		}
		images = append(images, data)
	}
	for _, u := range urls {
		data, err := h.service.downloadImage(r.Context(), u)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, fmt.Sprintf("failed to download image url: %v", err), "invalid_request_error", "bad_image_url")
			return nil, false
		}
		if len(data) > publicTaskMaxImageByte {
			httputil.WriteError(w, http.StatusBadRequest, "input image too large (max 25MB)", "invalid_request_error", "image_too_large")
			return nil, false
		}
		images = append(images, data)
	}
	if len(images) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "at least one of image_base64s or image_urls is required", "invalid_request_error", "missing_image")
		return nil, false
	}
	if len(images) > publicTaskMaxImages {
		httputil.WriteError(w, http.StatusBadRequest, "maximum 4 images allowed", "invalid_request_error", "too_many_images")
		return nil, false
	}
	return images, true
}

// GetTask handles GET /v1/images/tasks/{id}.
func (h *PublicTaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	auth, err := h.service.core.AuthenticateAny(r)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
		return
	}

	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
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

	if !publicTaskOwnedBy(task, auth) {
		httputil.WriteError(w, http.StatusNotFound, "Task not found", "invalid_request_error", "task_not_found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// publicTaskOwnedBy reports whether an API-key caller owns a task submitted via
// /v1/images/tasks. Tenant keys own only tasks they created (TenantKeyID set);
// member personal-key tasks carry the tenant id for billing but an empty
// TenantKeyID, so they belong to the user, not the tenant key.
func publicTaskOwnedBy(task *store.ImageTask, auth *proxy.AuthResult) bool {
	switch {
	case auth.IsSubUser():
		return task.SubUserID == auth.SubUser.ID
	case auth.IsTenant():
		return task.SubUserID == "" && task.TenantKeyID != "" && task.TenantID == auth.TenantKey.TenantID
	default:
		return task.ImageShareKeyID == nil && task.TenantKeyID == "" && task.SubUserID == "" && task.UserID == auth.User.ID
	}
}

// DeleteTask handles DELETE /v1/images/tasks/{id}. Lets an API-key caller
// delete its own task (e.g. one stuck pending). Processing tasks are refused
// with 409 so the worker never settles a charge against a deleted row.
func (h *PublicTaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	auth, err := h.service.core.AuthenticateAny(r)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
		return
	}

	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	taskID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid task ID", "invalid_request_error", "invalid_id")
		return
	}

	switch err := h.service.DeleteTaskPublic(r.Context(), auth, taskID); {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrTaskProcessing):
		httputil.WriteError(w, http.StatusConflict, "任务处理中，暂不可删除，请稍后重试", "conflict", "task_processing")
	case errors.Is(err, ErrTaskForbidden), errors.Is(err, ErrTaskNotFound):
		httputil.WriteError(w, http.StatusNotFound, "Task not found", "invalid_request_error", "task_not_found")
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to delete task", "server_error", "delete_failed")
	}
}

func writeTaskAccepted(w http.ResponseWriter, id int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": status})
}
