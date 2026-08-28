package image

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/balancer"
	"github.com/zhulang/llm-gateway/internal/bandwidth"
	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/pricing"
	"github.com/zhulang/llm-gateway/internal/proxy"
	"github.com/zhulang/llm-gateway/internal/router"
)

// Handler handles image generation and editing API requests.
type Handler struct {
	core         *proxy.Core
	pricingCache *pricing.Cache
	bwLimiter    *bandwidth.Limiter
}

// NewHandler creates a new image Handler.
func NewHandler(core *proxy.Core, pc *pricing.Cache, bwl *bandwidth.Limiter) *Handler {
	return &Handler{core: core, pricingCache: pc, bwLimiter: bwl}
}

// SetRouter atomically replaces the router (used during hot reload).
func (h *Handler) SetRouter(rt *router.Router) {
	h.core.SetRouter(rt)
}

// textToImageRequest is the request body for text-to-image generation.
type textToImageRequest struct {
	Prompt      string `json:"prompt"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Size        string `json:"size,omitempty"`
}

// imageEditRequest is the request body for image editing.
type imageEditRequest struct {
	Prompt       string   `json:"prompt"`
	ImageURLs    []string `json:"image_urls,omitempty"`
	ImageBase64s []string `json:"image_base64s,omitempty"`
	AspectRatio  string   `json:"aspect_ratio,omitempty"`
	Size         string   `json:"size,omitempty"`
}

// imageResponse is the response from the upstream API.
type imageResponse struct {
	ImageURLs []string `json:"image_urls"`
}

const (
	modelTextToImage = "gemini-3-pro-image-text-to-image"
	modelImageEdit   = "gemini-3-pro-image-edit"

	upstreamPathTextToImage = "/v3/gemini-3-pro-image-text-to-image"
	upstreamPathImageEdit   = "/v3/gemini-3-pro-image-edit"
)

// ServeTextToImage handles POST /v3/gemini-3-pro-image-text-to-image
func (h *Handler) ServeTextToImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	h.handleImageRequest(w, r, modelTextToImage, upstreamPathTextToImage)
}

// ServeImageEdit handles POST /v3/gemini-3-pro-image-edit
func (h *Handler) ServeImageEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	h.handleImageRequest(w, r, modelImageEdit, upstreamPathImageEdit)
}

// handleImageRequest is the common handler for both text-to-image and image-edit endpoints.
func (h *Handler) handleImageRequest(w http.ResponseWriter, r *http.Request, modelName, upstreamPath string) {
	// 1. Auth
	auth, err := h.core.AuthenticateAny(r)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
		return
	}

	if !auth.IsTenant() && !auth.IsSubUser() {
		r = r.WithContext(context.WithValue(r.Context(), admin.CtxUserIDKey, auth.User.ID))
	}

	// 2. Read body
	body, err := h.core.ReadBody(w, r)
	if err != nil {
		httputil.WriteError(w, http.StatusRequestEntityTooLarge, "Request body too large", "invalid_request_error", "request_too_large")
		return
	}

	// 3. Parse JSON to extract size for billing
	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid JSON in request body", "invalid_request_error", "invalid_json")
		return
	}

	size, _ := reqBody["size"].(string)
	if size == "" {
		size = "1K" // default
	}

	// 4. Model access check
	if err := h.core.CheckModelAccess(auth, modelName); err != nil {
		httputil.WriteError(w, http.StatusForbidden, err.Error(), "auth_error", "model_not_allowed")
		return
	}

	// 4.5 Balance check
	if err := h.core.CheckBilling(auth, modelName); err != nil {
		if proxy.IsNoPricingError(err) {
			httputil.WriteError(w, http.StatusForbidden, "Model billing not configured", "billing_error", "no_pricing")
		} else {
			httputil.WriteError(w, http.StatusPaymentRequired, "Insufficient balance", "billing_error", "insufficient_balance")
		}
		return
	}

	// 5. Route: get upstream from router
	rt := h.core.Router.Load()
	upstreams, _, found := rt.GetUpstreamsForTenant(auth.TenantID(), modelName)
	if !found {
		httputil.WriteError(w, http.StatusNotFound,
			fmt.Sprintf("Model %q not found", modelName),
			"invalid_request_error", "model_not_found")
		return
	}

	requestID := proxy.GetRequestID(r)

	// 6. Try upstreams with failover
	cfg := h.core.CfgHolder.Get()
	timeout := cfg.Server.RequestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	n := len(upstreams)
	tried := make(map[int]bool, n)
	start := h.core.Balancer.Counter()

	for len(tried) < n {
		var idx int
		foundUpstream := false
		for i := 0; i < n; i++ {
			idx = int((start + uint64(i)) % uint64(n))
			if tried[idx] {
				continue
			}
			if upstreams[idx].Breaker.AllowRequest() {
				foundUpstream = true
				break
			}
			tried[idx] = true
		}
		if !foundUpstream {
			break
		}
		tried[idx] = true
		upstream := &upstreams[idx]

		// Build upstream URL
		baseURL := strings.TrimRight(upstream.Config.BaseURL, "/")
		upstreamURL := baseURL + upstreamPath

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
		if err != nil {
			cancel()
			slog.Error("failed to create upstream request", "error", err)
			continue
		}

		upReq.Header.Set("Content-Type", "application/json")
		upReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)

		resp, err := h.core.Client.Do(upReq)
		if err != nil {
			cancel()
			if circuit.IsUpstreamFailure(err) {
				upstream.Breaker.RecordFailure()
			}
			slog.Warn("image upstream request failed", "upstream", upstream.Config.BaseURL, "error", err)
			continue
		}

		// Check for error status codes
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			upstream.Breaker.RecordFailure()
			slog.Warn("image upstream returned error status",
				"upstream", upstream.Config.BaseURL,
				"status", resp.StatusCode)
			w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
			resp.Body.Close()
			cancel()
			return
		}

		// Success — read response
		upstream.Breaker.RecordSuccess()
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		if err != nil {
			httputil.WriteError(w, http.StatusBadGateway, "Failed to read upstream response", "server_error", "upstream_read_error")
			return
		}

		// If upstream returned a non-2xx error (4xx), pass through
		if resp.StatusCode >= 400 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			w.Write(respBody)
			return
		}

		slog.Info("image request routed",
			"model", modelName,
			"upstream", upstream.Config.Provider,
			"status", resp.StatusCode,
		)

		// 7. Parse response to count images for billing
		var imgResp imageResponse
		imageCount := 1 // default to 1 if parsing fails
		if json.Unmarshal(respBody, &imgResp) == nil && len(imgResp.ImageURLs) > 0 {
			imageCount = len(imgResp.ImageURLs)
		}

		// 8. Calculate cost and charge
		go h.chargeForImages(auth, modelName, requestID, imageCount, size)

		// 9. Bandwidth queue: acquire slot before writing to client
		if h.bwLimiter != nil {
			release, bwErr := h.bwLimiter.Acquire(r.Context())
			if bwErr != nil {
				httputil.WriteError(w, http.StatusServiceUnavailable, "bandwidth queue timeout", "server_error", "bandwidth_timeout")
				return
			}
			defer release()
		}

		// 10. Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}

	httputil.WriteError(w, http.StatusServiceUnavailable,
		"All upstream providers are unavailable",
		"server_error", "upstream_unavailable")
}

// chargeForImages charges the auth entity for the given image count and size.
// Routes through AsyncChargeAuth so subscription quota is consumed first when applicable.
func (h *Handler) chargeForImages(auth *proxy.AuthResult, modelName, requestID string, imageCount int, size string) {
	usage := billing.UsageInfo{
		ImageSize:  size,
		ImageCount: imageCount,
	}
	h.core.AsyncChargeAuth(auth, modelName, requestID, usage)
}

// --- GPT Image API (OpenAI-compatible /v1/images/*) ---

// ServeGPTImageGenerations handles POST /v1/images/generations.
func (h *Handler) ServeGPTImageGenerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	h.handleGPTImageProxy(w, r, "/v1/images/generations")
}

// ServeGPTImageEdits handles POST /v1/images/edits.
func (h *Handler) ServeGPTImageEdits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	h.handleGPTImageProxy(w, r, "/v1/images/edits")
}

const gptImageMaxBodyBytes = 50 * 1024 * 1024 // 50MB

func (h *Handler) handleGPTImageProxy(w http.ResponseWriter, r *http.Request, upstreamPath string) {
	requestID := proxy.GetRequestID(r)

	auth, err := h.core.AuthenticateAny(r)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, gptImageMaxBodyBytes))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "failed to read request body", "invalid_request_error", "bad_request")
		return
	}

	// Detect content type to branch between JSON and multipart
	contentType := r.Header.Get("Content-Type")
	mediaType, params, _ := mime.ParseMediaType(contentType)

	var modelName string
	var sizeParam string
	var isStream bool
	var isMultipart bool
	var boundary string

	if mediaType == "multipart/form-data" {
		isMultipart = true
		boundary = params["boundary"]
		if boundary == "" {
			httputil.WriteError(w, http.StatusBadRequest, "missing boundary in multipart content-type", "invalid_request_error", "bad_request")
			return
		}
		modelName, err = extractModelFromMultipart(body, boundary)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "failed to parse multipart body: "+err.Error(), "invalid_request_error", "bad_request")
			return
		}
		sizeParam = extractSizeFromMultipart(body, boundary)
	} else {
		// JSON path (existing logic)
		var reqFields struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
			Size   string `json:"size"`
		}
		if err := json.Unmarshal(body, &reqFields); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid JSON body", "invalid_request_error", "bad_request")
			return
		}
		modelName = reqFields.Model
		isStream = reqFields.Stream
		sizeParam = reqFields.Size
	}

	if modelName == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "bad_request")
		return
	}

	rt := h.core.Router.Load()
	upstreams, modelInfo, found := rt.GetUpstreamsForTenant(auth.TenantID(), modelName)
	if !found {
		httputil.WriteError(w, http.StatusNotFound, fmt.Sprintf("model %q not found", modelName), "invalid_request_error", "model_not_found")
		return
	}

	canonicalModel := modelInfo.CanonicalName

	if err := h.core.CheckModelAccess(auth, canonicalModel); err != nil {
		httputil.WriteError(w, http.StatusForbidden, err.Error(), "auth_error", "model_not_allowed")
		return
	}

	if err := h.core.CheckBilling(auth, canonicalModel); err != nil {
		if proxy.IsNoPricingError(err) {
			httputil.WriteError(w, http.StatusForbidden, "model billing not configured", "billing_error", "no_pricing")
		} else {
			httputil.WriteError(w, http.StatusForbidden, "insufficient balance", "billing_error", "insufficient_balance")
		}
		return
	}

	// Validate size parameter for gpt-image-2
	if strings.Contains(canonicalModel, "gpt-image-2") {
		if msg := validateGPTImage2Size(sizeParam); msg != "" {
			httputil.WriteError(w, http.StatusBadRequest, msg, "invalid_request_error", "invalid_size")
			return
		}
	}

	// Strip unsupported input_fidelity parameter
	if isMultipart {
		body, contentType = stripInputFidelityFromMultipart(body, boundary)
		_, params, _ = mime.ParseMediaType(contentType)
		boundary = params["boundary"]
	} else {
		body = stripInputFidelityFromJSON(body)
	}

	// For multipart requests: preserve original Content-Type and handle model_override
	var extraHeaders http.Header
	upstreamBody := body
	if isMultipart {
		extraHeaders = http.Header{}
		extraHeaders.Set("Content-Type", contentType)
		if len(upstreams) > 0 && upstreams[0].Config.ModelOverride != "" {
			newBody, newCT := replaceModelInMultipart(body, boundary, upstreams[0].Config.ModelOverride)
			upstreamBody = newBody
			extraHeaders.Set("Content-Type", newCT)
		}
	}

	// Gemini native image edit: convert multipart to generateContent format
	if upstreamPath == "/v1/images/edits" && isMultipart && isGeminiNativeProvider(upstreams) {
		editParts, parseErr := parseMultipartForGemini(body, boundary)
		if parseErr != nil {
			httputil.WriteError(w, http.StatusBadRequest, "failed to parse image edit request: "+parseErr.Error(), "invalid_request_error", "bad_request")
			return
		}

		geminiRespBody, err := h.handleGeminiImageEdit(r.Context(), upstreams, editParts)
		if err != nil {
			slog.Error("gemini image edit failed", "model", canonicalModel, "error", err)
			httputil.WriteError(w, http.StatusBadGateway, "image edit failed: "+err.Error(), "upstream_error", "gemini_edit_failed")
			return
		}

		openAIBody, imageCount, convErr := geminiResponseToOpenAI(geminiRespBody)
		if convErr != nil {
			slog.Error("failed to convert gemini response", "model", canonicalModel, "error", convErr)
			httputil.WriteError(w, http.StatusBadGateway, "failed to process upstream response", "upstream_error", "response_parse_error")
			return
		}

		if h.bwLimiter != nil {
			release, bwErr := h.bwLimiter.Acquire(r.Context())
			if bwErr != nil {
				httputil.WriteError(w, http.StatusServiceUnavailable, "bandwidth queue timeout", "server_error", "bandwidth_timeout")
				return
			}
			defer release()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(openAIBody)

		usage := billing.UsageInfo{PromptTokens: 1, ImageCount: imageCount}
		h.core.AsyncChargeAuth(auth, canonicalModel, requestID, usage)
		return
	}

	result, err := h.core.Failover(r.Context(), upstreams, upstreamBody, http.MethodPost, upstreamPath, extraHeaders)
	if err != nil {
		slog.Error("all upstreams failed for image request", "model", canonicalModel, "error", err)
		httputil.WriteError(w, http.StatusBadGateway, "all upstreams failed", "upstream_error", "all_upstreams_failed")
		return
	}
	defer result.Cancel()
	defer result.Response.Body.Close()

	resp := result.Response

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}

	if isStream {
		h.relayGPTImageStream(w, resp, auth, canonicalModel, requestID)
	} else {
		h.relayGPTImageJSON(w, resp, auth, canonicalModel, requestID)
	}
}

func (h *Handler) relayGPTImageJSON(w http.ResponseWriter, resp *http.Response, auth *proxy.AuthResult, model, requestID string) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		httputil.WriteError(w, http.StatusBadGateway, "failed to read upstream response", "upstream_error", "read_error")
		return
	}

	if h.bwLimiter != nil {
		release, bwErr := h.bwLimiter.Acquire(context.Background())
		if bwErr != nil {
			httputil.WriteError(w, http.StatusServiceUnavailable, "bandwidth queue timeout", "server_error", "bandwidth_timeout")
			return
		}
		defer release()
	}

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	usage := extractGPTImageUsage(respBody)
	usage.ImageCount = countGPTImages(respBody)
	h.core.AsyncChargeAuth(auth, model, requestID, usage)
}

func (h *Handler) relayGPTImageStream(w http.ResponseWriter, resp *http.Response, auth *proxy.AuthResult, model, requestID string) {
	if h.bwLimiter != nil {
		release, bwErr := h.bwLimiter.Acquire(context.Background())
		if bwErr != nil {
			httputil.WriteError(w, http.StatusServiceUnavailable, "bandwidth queue timeout", "server_error", "bandwidth_timeout")
			return
		}
		defer release()
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.WriteError(w, http.StatusInternalServerError, "streaming not supported", "server_error", "no_flusher")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 10*1024*1024), 100*1024*1024) // 10MB initial, 100MB max

	var usage billing.UsageInfo
	imageCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		w.Write([]byte(line + "\n"))
		if line == "" {
			flusher.Flush()
		}
		if strings.HasPrefix(line, "data: ") {
			data := line[6:]
			if strings.Contains(data, "\"result\"") {
				imageCount++
			}
			if strings.Contains(data, "completed") {
				usage = extractGPTImageUsage([]byte(data))
			}
		}
	}
	flusher.Flush()

	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 {
		usage.PromptTokens = 1
	}
	if imageCount > 0 {
		usage.ImageCount = imageCount
	} else {
		usage.ImageCount = 1
	}
	h.core.AsyncChargeAuth(auth, model, requestID, usage)
}

func extractGPTImageUsage(data []byte) billing.UsageInfo {
	var parsed struct {
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(data, &parsed) == nil && parsed.Usage != nil {
		return billing.UsageInfo{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
		}
	}
	return billing.UsageInfo{PromptTokens: 1}
}

func countGPTImages(data []byte) int {
	var parsed struct {
		Data []json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &parsed) == nil && len(parsed.Data) > 0 {
		return len(parsed.Data)
	}
	return 1
}

// validateGPTImage2Size checks that size is valid for gpt-image-2.
// Rules: both edges must be multiples of 16, long/short ratio <= 3:1,
// total pixels in [655_360, 8_294_400]. "" and "auto" are always valid.
// Returns a user-friendly error string, or "" if valid.
func validateGPTImage2Size(size string) string {
	if size == "" || size == "auto" {
		return ""
	}
	parts := strings.SplitN(size, "x", 2)
	if len(parts) != 2 {
		return "invalid size format, expected WIDTHxHEIGHT (e.g. 1024x1024)"
	}
	var w, h int
	if _, err := fmt.Sscanf(parts[0], "%d", &w); err != nil || w <= 0 {
		return "invalid width in size parameter"
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &h); err != nil || h <= 0 {
		return "invalid height in size parameter"
	}
	if w > 3840 || h > 3840 {
		return "size exceeds maximum allowed edge of 3840px for gpt-image-2"
	}
	if w%16 != 0 || h%16 != 0 {
		return "width and height must both be multiples of 16 for gpt-image-2"
	}
	long, short := w, h
	if short > long {
		long, short = short, long
	}
	if long > 3*short {
		return "aspect ratio exceeds 3:1 limit for gpt-image-2"
	}
	pixels := w * h
	if pixels < 655_360 {
		return fmt.Sprintf("size %s is too small for gpt-image-2 (minimum 655,360 pixels, e.g. 1024x1024)", size)
	}
	if pixels > 8_294_400 {
		return fmt.Sprintf("size %s exceeds gpt-image-2 maximum of 8,294,400 total pixels", size)
	}
	return ""
}

// extractSizeFromMultipart reads the "size" text field from a multipart body.
// Returns "" if the field is absent or empty.
func extractSizeFromMultipart(body []byte, boundary string) string {
	r := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := r.NextPart()
		if err != nil {
			break
		}
		if part.FormName() == "size" {
			val, _ := io.ReadAll(io.LimitReader(part, 64))
			part.Close()
			return strings.TrimSpace(string(val))
		}
		part.Close()
	}
	return ""
}

// extractModelFromMultipart reads only the "model" text field from a multipart body.
func extractModelFromMultipart(body []byte, boundary string) (string, error) {
	r := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if part.FormName() == "model" {
			val, err := io.ReadAll(io.LimitReader(part, 256))
			part.Close()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(val)), nil
		}
		part.Close()
	}
	return "", fmt.Errorf("model field not found in multipart body")
}

// replaceModelInMultipart rewrites the multipart body with a new model value.
func replaceModelInMultipart(body []byte, boundary, newModel string) ([]byte, string) {
	r := multipart.NewReader(bytes.NewReader(body), boundary)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for {
		part, err := r.NextPart()
		if err != nil {
			break
		}
		if part.FormName() == "model" {
			fw, _ := w.CreateFormField("model")
			fw.Write([]byte(newModel))
		} else {
			pw, _ := w.CreatePart(part.Header)
			io.Copy(pw, part)
		}
		part.Close()
	}
	w.Close()
	return buf.Bytes(), w.FormDataContentType()
}

func stripInputFidelityFromMultipart(body []byte, boundary string) ([]byte, string) {
	r := multipart.NewReader(bytes.NewReader(body), boundary)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for {
		part, err := r.NextPart()
		if err != nil {
			break
		}
		if part.FormName() == "input_fidelity" {
			part.Close()
			continue
		}
		pw, _ := w.CreatePart(part.Header)
		io.Copy(pw, part)
		part.Close()
	}
	w.Close()
	return buf.Bytes(), w.FormDataContentType()
}

func stripInputFidelityFromJSON(body []byte) []byte {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return body
	}
	if _, ok := m["input_fidelity"]; !ok {
		return body
	}
	delete(m, "input_fidelity")
	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return out
}

func isGeminiNativeProvider(upstreams []balancer.Upstream) bool {
	if len(upstreams) == 0 {
		return false
	}
	return upstreams[0].Config.Provider == "google"
}

type geminiImagePart struct {
	Data     []byte
	MimeType string
}

type geminiEditParts struct {
	Prompt          string
	Images          []geminiImagePart
	Mask            []byte
	AspectRatio     string
	ImageResolution string
}

func parseMultipartForGemini(body []byte, boundary string) (*geminiEditParts, error) {
	r := multipart.NewReader(bytes.NewReader(body), boundary)
	parts := &geminiEditParts{}

	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading multipart: %w", err)
		}

		switch part.FormName() {
		case "prompt":
			val, _ := io.ReadAll(io.LimitReader(part, 10*1024))
			parts.Prompt = strings.TrimSpace(string(val))
		case "image", "image[]":
			data, _ := io.ReadAll(part)
			mimeType := part.Header.Get("Content-Type")
			if mimeType == "" {
				mimeType = http.DetectContentType(data)
			}
			parts.Images = append(parts.Images, geminiImagePart{Data: data, MimeType: mimeType})
		case "mask":
			parts.Mask, _ = io.ReadAll(part)
		case "aspect_ratio":
			val, _ := io.ReadAll(io.LimitReader(part, 64))
			parts.AspectRatio = strings.TrimSpace(string(val))
		case "image_resolution":
			val, _ := io.ReadAll(io.LimitReader(part, 64))
			parts.ImageResolution = strings.TrimSpace(string(val))
		}
		part.Close()
	}

	if parts.Prompt == "" {
		return nil, fmt.Errorf("prompt field is required")
	}
	if len(parts.Images) == 0 {
		return nil, fmt.Errorf("at least one image is required")
	}
	return parts, nil
}

func (h *Handler) handleGeminiImageEdit(ctx context.Context, upstreams []balancer.Upstream, parts *geminiEditParts) ([]byte, error) {
	geminiParts := []map[string]interface{}{
		{"text": parts.Prompt},
	}
	for _, img := range parts.Images {
		geminiParts = append(geminiParts, map[string]interface{}{
			"inlineData": map[string]interface{}{
				"mimeType": img.MimeType,
				"data":     base64.StdEncoding.EncodeToString(img.Data),
			},
		})
	}

	generationConfig := map[string]interface{}{
		"responseModalities": []string{"TEXT", "IMAGE"},
	}
	applyImageGenerationConfig(generationConfig, map[string]any{
		"aspect_ratio":     parts.AspectRatio,
		"image_resolution": parts.ImageResolution,
	})

	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": geminiParts},
		},
		"generationConfig": generationConfig,
	}

	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	cfg := h.core.CfgHolder.Get()
	timeout := cfg.Server.RequestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	for i, upstream := range upstreams {
		if !upstream.Breaker.AllowRequest() {
			continue
		}

		model := upstream.Config.ModelOverride
		if model == "" {
			model = upstream.Config.UpstreamName
		}

		baseURL := strings.TrimRight(upstream.Config.BaseURL, "/")
		upstreamURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent", baseURL, model)

		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, upstreamURL, bytes.NewReader(reqBody))
		if err != nil {
			cancel()
			continue
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)

		resp, err := h.core.Client.Do(httpReq)
		if err != nil {
			cancel()
			if circuit.IsUpstreamFailure(err) {
				upstreams[i].Breaker.RecordFailure()
			}
			slog.Warn("gemini image edit upstream failed", "upstream", upstream.Config.BaseURL, "error", err)
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			upstreams[i].Breaker.RecordFailure()
			slog.Warn("gemini image edit upstream error", "upstream", upstream.Config.BaseURL, "status", resp.StatusCode, "body", string(respBody))
			continue
		}

		if resp.StatusCode != http.StatusOK || readErr != nil {
			return nil, fmt.Errorf("gemini upstream returned status %d: %s", resp.StatusCode, string(respBody))
		}

		upstreams[i].Breaker.RecordSuccess()
		return respBody, nil
	}

	return nil, fmt.Errorf("all gemini upstreams failed for image edit")
}

func geminiResponseToOpenAI(respBody []byte) ([]byte, int, error) {
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData *struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData,omitempty"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, 0, fmt.Errorf("failed to parse gemini response: %w", err)
	}

	type imageData struct {
		B64JSON string `json:"b64_json"`
	}
	var images []imageData

	for _, candidate := range geminiResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && strings.HasPrefix(part.InlineData.MimeType, "image/") {
				images = append(images, imageData{B64JSON: part.InlineData.Data})
			}
		}
	}

	if len(images) == 0 {
		return nil, 0, fmt.Errorf("no images in gemini response")
	}

	openAIResp := struct {
		Created int64       `json:"created"`
		Data    []imageData `json:"data"`
	}{
		Created: time.Now().Unix(),
		Data:    images,
	}

	result, err := json.Marshal(openAIResp)
	return result, len(images), err
}
