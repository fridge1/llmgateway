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
)

// GenerationHandler handles image generation API requests.
type GenerationHandler struct {
	service     *Service
	shareStore  *imageshare.Store // optional; nil disables image-share path
}

// NewGenerationHandler creates a new GenerationHandler.
func NewGenerationHandler(service *Service) *GenerationHandler {
	return &GenerationHandler{
		service: service,
	}
}

// SetImageShareStore wires the image-share store so the handler can validate quotas
// and increment usage for image-share sessions. Pass nil to disable.
func (h *GenerationHandler) SetImageShareStore(s *imageshare.Store) {
	h.shareStore = s
}

// Generate handles POST /api/image/generate
func (h *GenerationHandler) Generate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	// Authenticate via JWT context
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	isShare := imageshare.IsImageShareRequest(r)

	// Parse request body
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalid_request_error", "invalid_json")
		return
	}

	// Validate request
	if !isShare && req.SessionID == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "session_id is required", "invalid_request_error", "missing_session_id")
		return
	}
	if !isShare && req.KeyID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "key_id is required", "invalid_request_error", "missing_key_id")
		return
	}
	if isShare {
		// Image-share callers don't supply or own session/key — derive them from the share key.
		req.KeyID = ""
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
		httputil.WriteError(w, http.StatusBadRequest, "size is required", "invalid_request_error", "missing_size")
		return
	}
	if req.N < 1 || req.N > 4 {
		httputil.WriteError(w, http.StatusBadRequest, "n must be between 1 and 4", "invalid_request_error", "invalid_n")
		return
	}

	// Validate size format
	_, _, err := h.service.ParseSize(req.Size)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_size")
		return
	}

	// Image-share path: ensure dedicated session and verify quota.
	var shareKeyID string
	if isShare {
		if h.shareStore == nil {
			httputil.WriteError(w, http.StatusServiceUnavailable, "image-share not configured", "server_error", "share_unavailable")
			return
		}
		shareKeyID = imageshare.KeyIDFromContext(r.Context())
		ownerID := imageshare.OwnerIDFromContext(r.Context())
		if shareKeyID == "" || ownerID == "" {
			httputil.WriteError(w, http.StatusUnauthorized, "missing share context", "auth_error", "missing_share_ctx")
			return
		}

		key, err := h.shareStore.GetKeyByID(shareKeyID)
		if err != nil {
			httputil.WriteError(w, http.StatusUnauthorized, "key revoked", "auth_error", "key_not_found")
			return
		}
		if key.Status != "active" || key.Remaining() < req.N {
			httputil.WriteError(w, http.StatusPaymentRequired, "图片生成次数不足，请联系管理员充值或更换密钥", "insufficient_quota", "quota_insufficient")
			return
		}

		// Ensure dedicated image session for this share key. Owner owns the row so
		// session.UserID == userID check inside service.Generate succeeds.
		sessionName := fmt.Sprintf("分享密钥-%s", key.KeyPrefix)
		sess, err := h.service.store.EnsureImageShareSession(r.Context(), ownerID, shareKeyID, sessionName)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to prepare session", "server_error", "session_failed")
			return
		}
		req.SessionID = sess.ID
	}

	// A tenant member's console request routes and bills through the tenant;
	// image-share requests stay on the owner's personal ledger.
	if !isShare {
		req.TenantID = h.service.core.MemberTenantIDForUser(userID)
	}

	// Generate images
	resp, err := h.service.Generate(r.Context(), userID, &req)
	if err != nil {
		// Check for service unavailable error
		if strings.Contains(err.Error(), "service unavailable") ||
			strings.Contains(err.Error(), "storage not configured") {
			httputil.WriteError(w, http.StatusServiceUnavailable,
				"图片生成服务暂时不可用，请联系管理员配置存储服务",
				"service_unavailable",
				"tos_not_configured")
			return
		}

		// Determine error type
		if err.Error() == "session not found" || err.Error() == "session does not belong to user" {
			httputil.WriteError(w, http.StatusNotFound, err.Error(), "invalid_request_error", "session_not_found")
			return
		}
		if strings.Contains(err.Error(), "insufficient balance") {
			httputil.WriteError(w, http.StatusPaymentRequired, "余额不足，请充值", "insufficient_balance", "insufficient_balance")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, extractUpstreamMessage(err), "server_error", "generation_failed")
		return
	}

	// Image-share path: increment quota_used by the actual successful image count.
	if isShare && shareKeyID != "" && h.shareStore != nil {
		actual := len(resp.ImageURLs)
		if actual > 0 {
			if _, err := h.shareStore.IncrementUsed(shareKeyID, actual); err != nil {
				// Concurrent over-spend or DB error: image was already generated and
				// owner was charged. We log and continue rather than fail the response.
				if errors.Is(err, imageshare.ErrQuotaExhausted) {
					slog.Warn("imageshare quota tracking lagged", "key_id", shareKeyID, "n", actual)
				} else {
					slog.Error("imageshare increment failed", "key_id", shareKeyID, "error", err)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetGenerations handles GET /api/image/sessions/:id/generations
func (h *GenerationHandler) GetGenerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	// Authenticate via JWT context
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	// Extract session ID from path
	sessionID, err := extractSessionID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid session ID", "invalid_request_error", "invalid_session_id")
		return
	}

	// Verify ownership
	session, err := h.service.store.GetImageSession(r.Context(), sessionID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "Session not found", "invalid_request_error", "session_not_found")
		return
	}
	if session.UserID != userID {
		httputil.WriteError(w, http.StatusForbidden, "Session does not belong to user", "auth_error", "not_authorized")
		return
	}

	// Parse query parameters
	limit := 10
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get generations
	generations, err := h.service.store.GetImageGenerations(r.Context(), sessionID, limit, offset)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to get generations", "server_error", "query_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(generations)
}

// Edit handles POST /api/image/edit (multipart/form-data)
func (h *GenerationHandler) Edit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	isShare := imageshare.IsImageShareRequest(r)

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid multipart form", "invalid_request_error", "invalid_form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	sessionID, _ := strconv.Atoi(r.FormValue("session_id"))
	keyID := r.FormValue("key_id")
	model := r.FormValue("model")
	prompt := r.FormValue("prompt")
	size := r.FormValue("size")
	n, _ := strconv.Atoi(r.FormValue("n"))

	if !isShare && sessionID == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "session_id is required", "invalid_request_error", "missing_session_id")
		return
	}
	if !isShare && keyID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "key_id is required", "invalid_request_error", "missing_key_id")
		return
	}
	if isShare {
		keyID = ""
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
		httputil.WriteError(w, http.StatusBadRequest, "prompt too long (max 32000 characters)", "invalid_request_error", "prompt_too_long")
		return
	}
	if size == "" {
		size = "1024x1024"
	}
	if n < 1 || n > 4 {
		n = 1
	}

	if _, _, err := h.service.ParseSize(size); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_size")
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

	// Image-share: ensure session and quota.
	var shareKeyID string
	if isShare {
		if h.shareStore == nil {
			httputil.WriteError(w, http.StatusServiceUnavailable, "image-share not configured", "server_error", "share_unavailable")
			return
		}
		shareKeyID = imageshare.KeyIDFromContext(r.Context())
		ownerID := imageshare.OwnerIDFromContext(r.Context())
		if shareKeyID == "" || ownerID == "" {
			httputil.WriteError(w, http.StatusUnauthorized, "missing share context", "auth_error", "missing_share_ctx")
			return
		}
		key, err := h.shareStore.GetKeyByID(shareKeyID)
		if err != nil {
			httputil.WriteError(w, http.StatusUnauthorized, "key revoked", "auth_error", "key_not_found")
			return
		}
		if key.Status != "active" || key.Remaining() < n {
			httputil.WriteError(w, http.StatusPaymentRequired, "图片生成次数不足，请联系管理员充值或更换密钥", "insufficient_quota", "quota_insufficient")
			return
		}
		sessionName := fmt.Sprintf("分享密钥-%s", key.KeyPrefix)
		sess, err := h.service.store.EnsureImageShareSession(r.Context(), ownerID, shareKeyID, sessionName)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to prepare session", "server_error", "session_failed")
			return
		}
		sessionID = sess.ID
	}

	req := &EditRequest{
		SessionID: sessionID,
		KeyID:     keyID,
		Model:     model,
		Prompt:    prompt,
		Size:      size,
		N:         n,
		Images:    images,
		Mask:      maskData,
	}
	if !isShare {
		req.TenantID = h.service.core.MemberTenantIDForUser(userID)
	}

	resp, err := h.service.Edit(r.Context(), userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "service unavailable") ||
			strings.Contains(err.Error(), "storage not configured") {
			httputil.WriteError(w, http.StatusServiceUnavailable,
				"图片编辑服务暂时不可用，请联系管理员配置存储服务",
				"service_unavailable", "tos_not_configured")
			return
		}
		if strings.Contains(err.Error(), "session not found") || strings.Contains(err.Error(), "session does not belong") {
			httputil.WriteError(w, http.StatusNotFound, err.Error(), "invalid_request_error", "session_not_found")
			return
		}
		if strings.Contains(err.Error(), "insufficient balance") {
			httputil.WriteError(w, http.StatusPaymentRequired, "余额不足，请充值", "insufficient_balance", "insufficient_balance")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, extractUpstreamMessage(err), "server_error", "edit_failed")
		return
	}

	if isShare && shareKeyID != "" && h.shareStore != nil {
		actual := len(resp.ImageURLs)
		if actual > 0 {
			if _, err := h.shareStore.IncrementUsed(shareKeyID, actual); err != nil {
				if errors.Is(err, imageshare.ErrQuotaExhausted) {
					slog.Warn("imageshare quota tracking lagged (edit)", "key_id", shareKeyID, "n", actual)
				} else {
					slog.Error("imageshare increment failed (edit)", "key_id", shareKeyID, "error", err)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
