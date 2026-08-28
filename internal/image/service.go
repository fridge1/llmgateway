package image

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/zhulang/llm-gateway/internal/balancer"
	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/imageshare"
	"github.com/zhulang/llm-gateway/internal/proxy"
	"github.com/zhulang/llm-gateway/internal/storage"
	"github.com/zhulang/llm-gateway/internal/store"
	"github.com/zhulang/llm-gateway/internal/subscription"
)

// Sentinel errors for task deletion flows.
var (
	ErrTaskNotFound   = errors.New("task not found")
	ErrTaskForbidden  = errors.New("task does not belong to user")
	ErrImageNotInTask = errors.New("image url not found in task results")
	ErrTaskProcessing = errors.New("task is processing and cannot be deleted")
)

// Service handles image generation business logic.
type Service struct {
	store               *store.PgStore
	billingService      *billing.BillingService
	subscriptionService *subscription.Service
	tosClient           *storage.TOSClient
	core                *proxy.Core
	imageShareStore     *imageshare.Store
	taskCh              chan struct{}
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	// upstreamSem caps total concurrent in-flight upstream image calls across all
	// async workers. nil means no gating.
	upstreamSem chan struct{}
}

// NewService creates a new image Service.
// upstreamConcurrency caps total concurrent in-flight upstream image calls; <=0 means no gate.
func NewService(s *store.PgStore, bs *billing.BillingService, subSvc *subscription.Service, tos *storage.TOSClient, core *proxy.Core, upstreamConcurrency int) *Service {
	var sem chan struct{}
	if upstreamConcurrency > 0 {
		sem = make(chan struct{}, upstreamConcurrency)
	}
	return &Service{
		store:               s,
		billingService:      bs,
		subscriptionService: subSvc,
		tosClient:           tos,
		core:                core,
		upstreamSem:         sem,
	}
}

// SetImageShareStore wires the image-share store so the worker can decrement quotas
// for tasks owned by an image-share key. Pass nil to disable.
func (s *Service) SetImageShareStore(store *imageshare.Store) {
	s.imageShareStore = store
}

// acquireUpstreamSlot blocks until a global upstream slot is free or ctx is done.
// When the gate is disabled (nil), it returns immediately.
func (s *Service) acquireUpstreamSlot(ctx context.Context) error {
	if s.upstreamSem == nil {
		return nil
	}
	select {
	case s.upstreamSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseUpstreamSlot frees a previously acquired global slot. Safe to call
// when the gate is disabled.
func (s *Service) releaseUpstreamSlot() {
	if s.upstreamSem == nil {
		return
	}
	<-s.upstreamSem
}

// chargeUser routes a user-side charge through subscription quota first; falls back to balance.
// Returns (cost, error). When subscription absorbs the charge, cost is 0.
func (s *Service) chargeUser(userID, model, requestID string, usage billing.UsageInfo, apiKeyID string) (float64, error) {
	if s.subscriptionService != nil {
		result, err := s.subscriptionService.CheckAccess(userID, model)
		if err == nil && result.Covered && result.WithinQuota {
			if recErr := s.subscriptionService.RecordUsage(userID, model, requestID, usage, apiKeyID); recErr != nil {
				slog.Error("subscription usage record failed", "user_id", userID, "model", model, "error", recErr)
			}
			return 0, nil
		}
	}
	return s.billingService.ChargeAndReturnCost(userID, model, requestID, usage, apiKeyID)
}

// checkBalance accepts the request if subscription covers the model and is within quota; otherwise checks balance.
// checkBalance verifies the caller can afford the request. When tenantID is
// non-empty (the user is a tenant member) it applies tenant billing semantics
// (owner subscription first, then tenant balance) via the same gate API-key
// requests use; otherwise it checks the personal subscription and balance.
func (s *Service) checkBalance(userID, tenantID, model string) error {
	if tenantID != "" {
		return s.core.CheckBilling(&proxy.AuthResult{User: &store.User{ID: userID}, MemberTenantID: tenantID}, model)
	}
	if s.subscriptionService != nil {
		if r, err := s.subscriptionService.CheckAccess(userID, model); err == nil && r.Covered && r.WithinQuota {
			return nil
		}
	}
	return s.billingService.CheckBalance(userID, model)
}

// GenerateRequest represents an image generation request.
type GenerateRequest struct {
	SessionID int            `json:"session_id"`
	KeyID     string         `json:"key_id"`
	Model     string         `json:"model"`
	Prompt    string         `json:"prompt"`
	Size      string         `json:"size"`
	N         int            `json:"n"`
	Params    map[string]any `json:"params,omitempty"`
	// TenantID, when non-empty, routes to the tenant's upstream overrides and
	// bills the tenant. Filled from the persisted task identity on the async
	// worker path, or from the caller's tenant membership on JWT console
	// requests; empty for image-share requests.
	TenantID string `json:"-"`
}

// GenerateResponse represents an image generation response.
type GenerateResponse struct {
	ID        int      `json:"id"`
	ImageURLs []string `json:"image_urls"`
	Cost      float64  `json:"cost"`
	CreatedAt string   `json:"created_at"`
}

// ParseSize validates a "WIDTHxHEIGHT" string against constraints accepted
// by current OpenAI image models (notably gpt-image-2):
//   - both edges in [256, 3840]
//   - both edges multiples of 16
//   - long/short ratio <= 3:1
//   - total pixels in [655_360, 8_294_400]
//
// "auto" / "" are treated as valid (the upstream picks the size); width and
// height are returned as 0 in that case.
func (s *Service) ParseSize(size string) (int, int, error) {
	if size == "" || size == "auto" {
		return 0, 0, nil
	}
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid size format, expected WIDTHxHEIGHT")
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid width: %w", err)
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height: %w", err)
	}

	if width < 256 || width > 3840 || height < 256 || height > 3840 {
		return 0, 0, fmt.Errorf("width and height must be between 256 and 3840")
	}
	if width%16 != 0 || height%16 != 0 {
		return 0, 0, fmt.Errorf("width and height must be multiples of 16")
	}
	long, short := width, height
	if short > long {
		long, short = short, long
	}
	if long > 3*short {
		return 0, 0, fmt.Errorf("long/short ratio must be <= 3:1")
	}
	pixels := width * height
	if pixels < 655_360 || pixels > 8_294_400 {
		return 0, 0, fmt.Errorf("total pixels must be between 655,360 and 8,294,400")
	}

	return width, height, nil
}

// Generate handles the complete image generation flow.
func (s *Service) Generate(ctx context.Context, userID string, req *GenerateRequest) (*GenerateResponse, error) {
	// 0. 检查 TOS 客户端是否可用
	if s.tosClient == nil {
		return nil, fmt.Errorf("image generation service unavailable: storage not configured")
	}

	// 1. 验证会话
	session, err := s.store.GetImageSession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("session does not belong to user")
	}

	// 2. 解析尺寸（仅用于校验）
	if _, _, err := s.ParseSize(req.Size); err != nil {
		return nil, err
	}

	// 3. 确定计费档位（Gemini 模型实际分辨率在 params.image_resolution 中）
	imageSize := resolveImageSize(req.Params, req.Size, s.tosClient)

	// 4. 检查余额（预估费用）
	if err := s.checkBalance(userID, req.TenantID, req.Model); err != nil {
		return nil, fmt.Errorf("insufficient balance: %w", err)
	}

	// 5. 顺序调用上游 API 生成多张图片
	// 上游 API 不支持一次返回多张，逐次请求避免并发限流
	var upstreamURLs []string
	singleReq := &GenerateRequest{
		SessionID: req.SessionID,
		KeyID:     req.KeyID,
		Model:     req.Model,
		Prompt:    req.Prompt,
		Size:      req.Size,
		N:         1, // 每次只请求 1 张
		Params:    req.Params,
		TenantID:  req.TenantID,
	}

	for i := 0; i < req.N; i++ {
		var urls []string
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
			urls, lastErr = s.callUpstreamAPI(ctx, singleReq, userID)
			if lastErr == nil {
				break
			}
			slog.Warn("image generation attempt failed, retrying", "image", i+1, "attempt", attempt+1, "error", lastErr)
		}
		if lastErr != nil {
			slog.Warn("image generation failed after retries", "image", i+1, "total", req.N, "error", lastErr)
			continue
		}
		if len(urls) > 0 {
			upstreamURLs = append(upstreamURLs, urls[0])
		}
	}

	if len(upstreamURLs) == 0 {
		return nil, fmt.Errorf("all image generation attempts failed")
	}

	// 6. 下载并上传到 TOS
	var tosURLs []string
	for _, url := range upstreamURLs {
		if s.tosClient.IsTOSURL(url) {
			tosURLs = append(tosURLs, url)
			continue
		}
		// 下载图片
		imageData, err := s.downloadImage(ctx, url)
		if err != nil {
			// Fallback to upstream URL
			tosURLs = append(tosURLs, url)
			continue
		}

		// Upload to TOS
		tosURL, err := s.tosClient.UploadImage(ctx, imageData, userID)
		if err != nil {
			// Fallback to upstream URL
			tosURLs = append(tosURLs, url)
			continue
		}

		tosURLs = append(tosURLs, tosURL)
	}

	// 7. 计算费用并扣费（统一通过 billingService 查询定价）
	actualImageCount := len(tosURLs)
	usage := billing.UsageInfo{
		ImageSize:  imageSize,
		ImageCount: actualImageCount,
	}

	// Generate a unique request ID for billing
	requestID := fmt.Sprintf("img-%d-%d", session.ID, time.Now().UnixNano())

	var cost float64
	if req.TenantID != "" {
		cost, err = s.core.ChargeTenantSync(req.TenantID, "", req.Model, requestID, usage)
	} else {
		cost, err = s.chargeUser(userID, req.Model, requestID, usage, "")
	}
	if err != nil {
		return nil, fmt.Errorf("billing failed: %w", err)
	}

	// 8. 保存生成记录
	gen := &store.ImageGeneration{
		SessionID:  req.SessionID,
		UserID:     userID,
		Model:      req.Model,
		Prompt:     req.Prompt,
		Size:       req.Size,
		ImageCount: actualImageCount,
		ImageURLs:  tosURLs,
		Cost:       cost,
	}

	gen, err = s.store.CreateImageGeneration(ctx, gen)
	if err != nil {
		// Log error - billing already happened
		return nil, fmt.Errorf("failed to save generation record: %w", err)
	}

	// 9. 返回结果
	return &GenerateResponse{
		ID:        gen.ID,
		ImageURLs: gen.ImageURLs,
		Cost:      gen.Cost,
		CreatedAt: gen.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// callUpstreamAPI calls the upstream image generation API.
// For Google (Gemini) upstreams it uses the native generateContent format;
// for others it uses the OpenAI /v1/images/generations format.
func (s *Service) callUpstreamAPI(ctx context.Context, req *GenerateRequest, userID string) ([]string, error) {
	// 获取路由信息
	rt := s.core.Router.Load()
	upstreams, _, found := rt.GetUpstreamsForTenant(req.TenantID, req.Model)
	if !found {
		return nil, fmt.Errorf("model %q not found", req.Model)
	}

	// 构建 OpenAI 兼容的请求（仅非 Google upstream 使用）
	var openAIBody []byte

	for _, upstream := range upstreams {
		if !upstream.Breaker.AllowRequest() {
			continue
		}

		// Gemini 原生路径
		if upstream.Config.Provider == "google" {
			urls, err := s.callGeminiImageAPI(ctx, &upstream, req, userID)
			if err != nil {
				slog.Warn("gemini image upstream failed", "upstream", upstream.Config.BaseURL, "error", err)
				continue
			}
			return urls, nil
		}

		// OpenAI 兼容路径
		if openAIBody == nil {
			payload := map[string]interface{}{
				"model":  req.Model,
				"prompt": req.Prompt,
				"size":   req.Size,
				"n":      req.N,
			}
			// "size" in params overrides req.Size so that non-WxH values
			// (e.g. "auto") that the frontend stashes in params are forwarded
			// correctly to the upstream instead of the billing-tier fallback.
			for _, k := range []string{"quality", "size"} {
				if v, ok := req.Params[k]; ok {
					if s, ok := v.(string); !ok || s != "" {
						payload[k] = v
					}
				}
			}
			body, err := json.Marshal(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request: %w", err)
			}
			openAIBody = body
		}

		baseURL := strings.TrimRight(upstream.Config.BaseURL, "/")
		upstreamURL := baseURL + "/v1/images/generations"

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(openAIBody))
		if err != nil {
			continue
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)

		resp, err := s.core.Client.Do(httpReq)
		if err != nil {
			if circuit.IsUpstreamFailure(err) {
				upstream.Breaker.RecordFailure()
			}
			continue
		}

		if resp.StatusCode >= 500 {
			upstream.Breaker.RecordFailure()
			resp.Body.Close()
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			upstream.Breaker.RecordFailure()
			resp.Body.Close()
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("upstream returned status %d: %s", resp.StatusCode, string(respBody))
		}

		upstream.Breaker.RecordSuccess()

		var urls []string

		var imgResp struct {
			Data []struct {
				URL     string `json:"url"`
				B64JSON string `json:"b64_json"`
			} `json:"data"`
		}

		if err := json.Unmarshal(respBody, &imgResp); err == nil && len(imgResp.Data) > 0 {
			for _, img := range imgResp.Data {
				if img.URL != "" {
					urls = append(urls, img.URL)
				} else if img.B64JSON != "" {
					imageData, err := base64.StdEncoding.DecodeString(img.B64JSON)
					if err != nil {
						slog.Warn("failed to decode b64_json from upstream", "error", err)
						continue
					}
					tosURL, err := s.tosClient.UploadImage(ctx, imageData, userID)
					if err != nil {
						slog.Warn("failed to upload b64_json image to TOS", "error", err)
						continue
					}
					urls = append(urls, tosURL)
				}
			}
		} else {
			var altResp struct {
				ImageURLs []string `json:"image_urls"`
			}
			if err := json.Unmarshal(respBody, &altResp); err == nil && len(altResp.ImageURLs) > 0 {
				urls = altResp.ImageURLs
			}
		}

		if len(urls) == 0 {
			return nil, fmt.Errorf("no images returned from upstream")
		}

		return urls, nil
	}

	return nil, fmt.Errorf("all upstreams failed")
}

// callGeminiImageAPI calls the Gemini native generateContent API for image generation.
func (s *Service) callGeminiImageAPI(ctx context.Context, upstream *balancer.Upstream, req *GenerateRequest, userID string) ([]string, error) {
	model := req.Model
	if upstream.Config.ModelOverride != "" {
		model = upstream.Config.ModelOverride
	}

	generationConfig := map[string]interface{}{
		"responseModalities": []string{"TEXT", "IMAGE"},
	}
	applyImageGenerationConfig(generationConfig, req.Params)

	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": req.Prompt},
				},
			},
		},
		"generationConfig": generationConfig,
	}

	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	baseURL := strings.TrimRight(upstream.Config.BaseURL, "/")
	upstreamURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent", baseURL, model)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)

	resp, err := s.core.Client.Do(httpReq)
	if err != nil {
		if circuit.IsUpstreamFailure(err) {
			upstream.Breaker.RecordFailure()
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		upstream.Breaker.RecordFailure()
		return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned status %d: %s", resp.StatusCode, string(respBody))
	}

	upstream.Breaker.RecordSuccess()

	// 解析 Gemini generateContent 响应，提取 inlineData 中的 base64 图片
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
		return nil, fmt.Errorf("failed to parse gemini response: %w", err)
	}

	var urls []string
	for _, candidate := range geminiResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData == nil || !strings.HasPrefix(part.InlineData.MimeType, "image/") {
				continue
			}
			imageData, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
			if err != nil {
				slog.Warn("failed to decode gemini image base64", "error", err)
				continue
			}
			tosURL, err := s.tosClient.UploadImage(ctx, imageData, userID)
			if err != nil {
				slog.Warn("failed to upload gemini image to TOS", "error", err)
				continue
			}
			urls = append(urls, tosURL)
		}
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no images returned from gemini upstream")
	}

	return urls, nil
}

// downloadImage downloads an image from a URL
func (s *Service) downloadImage(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.core.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// EditRequest represents an image edit request.
type EditRequest struct {
	SessionID int
	KeyID     string
	Model     string
	Prompt    string
	Size      string
	N         int
	Params    map[string]any
	Images    [][]byte
	Mask      []byte
	// TenantID, when non-empty, routes to the tenant's upstream overrides and
	// bills the tenant (async worker path or JWT console requests from tenant
	// members; empty for image-share requests).
	TenantID string
}

// Edit handles the complete image editing flow.
func (s *Service) Edit(ctx context.Context, userID string, req *EditRequest) (*GenerateResponse, error) {
	if s.tosClient == nil {
		return nil, fmt.Errorf("image generation service unavailable: storage not configured")
	}

	session, err := s.store.GetImageSession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("session does not belong to user")
	}

	if _, _, err := s.ParseSize(req.Size); err != nil {
		return nil, err
	}

	imageSize := resolveImageSize(req.Params, req.Size, s.tosClient)

	if err := s.checkBalance(userID, req.TenantID, req.Model); err != nil {
		return nil, fmt.Errorf("insufficient balance: %w", err)
	}

	// Build multipart body once, reuse bytes for retries
	multipartBody, contentType, err := s.buildEditMultipart(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	var allImageData [][]byte
	for i := 0; i < req.N; i++ {
		var imgData [][]byte
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
			imgData, lastErr = s.callUpstreamEditAPI(ctx, req, multipartBody, contentType)
			if lastErr == nil {
				break
			}
			slog.Warn("image edit attempt failed, retrying", "image", i+1, "attempt", attempt+1, "error", lastErr)
		}
		if lastErr != nil {
			slog.Warn("image edit failed after retries", "image", i+1, "total", req.N, "error", lastErr)
			continue
		}
		allImageData = append(allImageData, imgData...)
	}

	if len(allImageData) == 0 {
		return nil, fmt.Errorf("all image edit attempts failed")
	}

	// PLACEHOLDER_EDIT_TOS_BILLING

	// Upload to TOS
	var tosURLs []string
	for _, data := range allImageData {
		tosURL, err := s.tosClient.UploadImage(ctx, data, userID)
		if err != nil {
			continue
		}
		tosURLs = append(tosURLs, tosURL)
	}

	if len(tosURLs) == 0 {
		return nil, fmt.Errorf("failed to upload edited images")
	}

	actualImageCount := len(tosURLs)
	usage := billing.UsageInfo{
		ImageSize:  imageSize,
		ImageCount: actualImageCount,
	}

	requestID := fmt.Sprintf("img-edit-%d-%d", session.ID, time.Now().UnixNano())

	var cost float64
	if req.TenantID != "" {
		cost, err = s.core.ChargeTenantSync(req.TenantID, "", req.Model, requestID, usage)
	} else {
		cost, err = s.chargeUser(userID, req.Model, requestID, usage, "")
	}
	if err != nil {
		return nil, fmt.Errorf("billing failed: %w", err)
	}

	gen := &store.ImageGeneration{
		SessionID:  req.SessionID,
		UserID:     userID,
		Model:      req.Model,
		Prompt:     req.Prompt,
		Size:       req.Size,
		ImageCount: actualImageCount,
		ImageURLs:  tosURLs,
		Cost:       cost,
	}

	gen, err = s.store.CreateImageGeneration(ctx, gen)
	if err != nil {
		return nil, fmt.Errorf("failed to save generation record: %w", err)
	}

	return &GenerateResponse{
		ID:        gen.ID,
		ImageURLs: gen.ImageURLs,
		Cost:      gen.Cost,
		CreatedAt: gen.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// buildEditMultipart builds the multipart body for an edit request.
func (s *Service) buildEditMultipart(req *EditRequest) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	w.WriteField("model", req.Model)
	w.WriteField("prompt", req.Prompt)
	w.WriteField("size", req.Size)
	w.WriteField("n", "1")

	for _, k := range []string{"quality"} {
		if v, ok := req.Params[k]; ok {
			if sv, ok := v.(string); ok && sv != "" {
				w.WriteField(k, sv)
			}
		}
	}

	for i, imgData := range req.Images {
		fieldName := "image"
		if len(req.Images) > 1 {
			fieldName = "image[]"
		}
		// 检测实际的图片格式，避免 Content-Type 与文件头不匹配
		ext := detectImageExtension(imgData)
		part, err := w.CreateFormFile(fieldName, fmt.Sprintf("image_%d%s", i, ext))
		if err != nil {
			return nil, "", err
		}
		part.Write(imgData)
	}

	if len(req.Mask) > 0 {
		ext := detectImageExtension(req.Mask)
		part, err := w.CreateFormFile("mask", "mask"+ext)
		if err != nil {
			return nil, "", err
		}
		part.Write(req.Mask)
	}

	w.Close()
	return buf.Bytes(), w.FormDataContentType(), nil
}

// detectImageExtension 根据文件头检测图片格式并返回对应的扩展名
func detectImageExtension(data []byte) string {
	mimeType := http.DetectContentType(data)
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".png" // 默认使用 png
	}
}

// PLACEHOLDER_UPSTREAM_EDIT

// callUpstreamEditAPI dispatches an image edit request to upstreams, picking the
// right protocol per upstream: Google's generateContent for provider=google,
// OpenAI multipart /v1/images/edits otherwise.
func (s *Service) callUpstreamEditAPI(ctx context.Context, req *EditRequest, multipartBody []byte, contentType string) ([][]byte, error) {
	rt := s.core.Router.Load()
	upstreams, _, found := rt.GetUpstreamsForTenant(req.TenantID, req.Model)
	if !found {
		return nil, fmt.Errorf("model %q not found", req.Model)
	}

	var geminiReqBody []byte // lazily built, shared across google upstreams within one call

	for _, upstream := range upstreams {
		if !upstream.Breaker.AllowRequest() {
			slog.Warn("image edit upstream skipped (breaker open)",
				"model", req.Model, "upstream", upstream.Config.BaseURL)
			continue
		}

		// Google (Gemini) native path: convert to generateContent JSON.
		if upstream.Config.Provider == "google" {
			if geminiReqBody == nil {
				body, err := buildGeminiEditBody(req)
				if err != nil {
					return nil, fmt.Errorf("failed to build gemini edit request: %w", err)
				}
				geminiReqBody = body
			}
			imgs, err := s.callGeminiEditUpstream(ctx, &upstream, req.Model, geminiReqBody)
			if err != nil {
				slog.Warn("image edit upstream failed (gemini)",
					"model", req.Model, "upstream", upstream.Config.BaseURL, "error", err)
				continue
			}
			return imgs, nil
		}

		// OpenAI-compatible multipart path.
		baseURL := strings.TrimRight(upstream.Config.BaseURL, "/")
		upstreamURL := baseURL + "/v1/images/edits"

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(multipartBody))
		if err != nil {
			slog.Warn("image edit upstream skipped (build request)",
				"model", req.Model, "upstream", upstream.Config.BaseURL, "error", err)
			continue
		}

		httpReq.Header.Set("Content-Type", contentType)
		httpReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)

		resp, err := s.core.Client.Do(httpReq)
		if err != nil {
			if circuit.IsUpstreamFailure(err) {
				upstream.Breaker.RecordFailure()
			}
			slog.Warn("image edit upstream transport error",
				"model", req.Model, "upstream", upstream.Config.BaseURL, "error", err)
			continue
		}

		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			upstream.Breaker.RecordFailure()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024))
			resp.Body.Close()
			slog.Warn("image edit upstream returned retryable status",
				"model", req.Model, "upstream", upstream.Config.BaseURL,
				"status", resp.StatusCode, "body", string(body))
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			slog.Warn("image edit upstream read body failed",
				"model", req.Model, "upstream", upstream.Config.BaseURL,
				"status", resp.StatusCode, "error", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("upstream returned status %d: %s", resp.StatusCode, string(respBody))
		}

		upstream.Breaker.RecordSuccess()
		return s.parseEditResponse(ctx, respBody)
	}

	return nil, fmt.Errorf("all upstreams failed")
}

// applyImageGenerationConfig writes Gemini image generation hints into
// generationConfig under both `imageConfig` (legacy / Google's public REST) and
// `responseFormat.image` (newer SDK exposure). Different relays accept
// different field names, sending both is the most compatible option.
func applyImageGenerationConfig(generationConfig map[string]interface{}, params map[string]any) {
	imageCfg := map[string]interface{}{}
	if v, ok := params["aspect_ratio"]; ok {
		if sv, ok := v.(string); ok && sv != "" {
			imageCfg["aspectRatio"] = sv
		}
	}
	if v, ok := params["image_resolution"]; ok {
		if sv, ok := v.(string); ok && sv != "" {
			imageCfg["imageSize"] = sv
		}
	}
	if len(imageCfg) == 0 {
		return
	}
	generationConfig["imageConfig"] = imageCfg
	generationConfig["responseFormat"] = map[string]interface{}{"image": imageCfg}
}

// buildGeminiEditBody converts an EditRequest to a Gemini generateContent JSON body.
func buildGeminiEditBody(req *EditRequest) ([]byte, error) {
	parts := []map[string]interface{}{
		{"text": req.Prompt},
	}
	for _, img := range req.Images {
		mimeType := http.DetectContentType(img)
		if !strings.HasPrefix(mimeType, "image/") {
			mimeType = "image/png"
		}
		parts = append(parts, map[string]interface{}{
			"inlineData": map[string]interface{}{
				"mimeType": mimeType,
				"data":     base64.StdEncoding.EncodeToString(img),
			},
		})
	}

	generationConfig := map[string]interface{}{
		"responseModalities": []string{"TEXT", "IMAGE"},
	}
	applyImageGenerationConfig(generationConfig, req.Params)

	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": parts},
		},
		"generationConfig": generationConfig,
	}
	return json.Marshal(geminiReq)
}

// callGeminiEditUpstream POSTs a generateContent request to one Google upstream
// and returns decoded image bytes from inlineData parts.
func (s *Service) callGeminiEditUpstream(ctx context.Context, upstream *balancer.Upstream, reqModel string, body []byte) ([][]byte, error) {
	model := reqModel
	if upstream.Config.ModelOverride != "" {
		model = upstream.Config.ModelOverride
	}

	baseURL := strings.TrimRight(upstream.Config.BaseURL, "/")
	upstreamURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent", baseURL, model)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)

	resp, err := s.core.Client.Do(httpReq)
	if err != nil {
		if circuit.IsUpstreamFailure(err) {
			upstream.Breaker.RecordFailure()
		}
		return nil, fmt.Errorf("transport: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		upstream.Breaker.RecordFailure()
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024))
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(preview))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

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
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var images [][]byte
	for _, candidate := range geminiResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData == nil || !strings.HasPrefix(part.InlineData.MimeType, "image/") {
				continue
			}
			data, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
			if err != nil {
				slog.Warn("gemini edit: decode inline data failed", "error", err)
				continue
			}
			images = append(images, data)
		}
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no images in gemini response")
	}

	upstream.Breaker.RecordSuccess()
	return images, nil
}

// parseEditResponse handles both b64_json and url response formats.
func (s *Service) parseEditResponse(ctx context.Context, respBody []byte) ([][]byte, error) {
	var resp struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse upstream response: %w", err)
	}

	var images [][]byte
	for _, item := range resp.Data {
		if item.B64JSON != "" {
			data, err := base64.StdEncoding.DecodeString(item.B64JSON)
			if err != nil {
				continue
			}
			images = append(images, data)
		} else if item.URL != "" {
			data, err := s.downloadImage(ctx, item.URL)
			if err != nil {
				continue
			}
			images = append(images, data)
		}
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no images in upstream response")
	}
	return images, nil
}

// SubmitGeneratePublic creates a pending generate task for an API-key caller,
// persisting the billing identity (user / tenant / sub-user) so the detached
// worker can settle against the right entity. It does NOT touch image-share.
func (s *Service) SubmitGeneratePublic(ctx context.Context, auth *proxy.AuthResult, model, prompt, size string, n int, params map[string]any) (*store.ImageTask, error) {
	if s.tosClient == nil {
		return nil, fmt.Errorf("image generation service unavailable: storage not configured")
	}
	if _, _, err := s.ParseSize(size); err != nil {
		return nil, err
	}
	if err := s.core.CheckBilling(auth, model); err != nil {
		return nil, fmt.Errorf("insufficient balance: %w", err)
	}
	task := &store.ImageTask{
		Type:       "generate",
		Model:      model,
		Prompt:     prompt,
		Size:       size,
		ImageCount: n,
		Params:     params,
	}
	applyBillingIdentity(task, auth)
	task, err := s.store.CreateImageTask(ctx, task)
	if err != nil {
		return nil, err
	}
	s.NotifyNewTask()
	return task, nil
}

// SubmitEditPublic is the edit counterpart of SubmitGeneratePublic. images are
// raw bytes already assembled from base64 and/or downloaded URLs by the handler.
func (s *Service) SubmitEditPublic(ctx context.Context, auth *proxy.AuthResult, model, prompt, size string, n int, images [][]byte, mask []byte, params map[string]any) (*store.ImageTask, error) {
	if s.tosClient == nil {
		return nil, fmt.Errorf("image generation service unavailable: storage not configured")
	}
	if _, _, err := s.ParseSize(size); err != nil {
		return nil, err
	}
	if err := s.core.CheckBilling(auth, model); err != nil {
		return nil, fmt.Errorf("insufficient balance: %w", err)
	}
	var b64Images []string
	for _, img := range images {
		b64Images = append(b64Images, base64.StdEncoding.EncodeToString(img))
	}
	task := &store.ImageTask{
		Type:        "edit",
		Model:       model,
		Prompt:      prompt,
		Size:        size,
		ImageCount:  n,
		Params:      params,
		InputImages: b64Images,
		InputMask:   mask,
	}
	applyBillingIdentity(task, auth)
	task, err := s.store.CreateImageTask(ctx, task)
	if err != nil {
		return nil, err
	}
	s.NotifyNewTask()
	return task, nil
}

// applyBillingIdentity fills the task's UserID and billing-identity columns from
// an authenticated API key. UserID doubles as the TOS path prefix and the
// ownership anchor; for tenant/sub-user keys it holds the tenant/sub-user id so
// it never collides with a real console user's task list (which filters by the
// caller's own user id).
func applyBillingIdentity(task *store.ImageTask, auth *proxy.AuthResult) {
	switch {
	case auth.IsSubUser():
		task.UserID = auth.SubUser.ID
		task.SubUserID = auth.SubUser.ID
		task.SubUserKeyID = auth.SubUserKey.ID
		task.TenantID = auth.SubUserKey.TenantID
	case auth.IsTenant():
		task.UserID = auth.TenantKey.TenantID
		task.TenantID = auth.TenantKey.TenantID
		task.TenantKeyID = auth.TenantKey.ID
	default:
		task.UserID = auth.User.ID
		task.APIKeyID = auth.APIKeyID
		// A personal key whose owner belongs to a tenant routes and bills through
		// that tenant. UserID stays the real user id (TOS prefix + ownership anchor)
		// and TenantKeyID is left empty so the task is not mistaken for a tenant-key
		// task in ownership checks; settleTaskCharge keys off TenantID for billing.
		if auth.MemberTenantID != "" {
			task.TenantID = auth.MemberTenantID
		}
	}
}

// settleTaskCharge charges the entity that owns the task and returns the cost.
// Routing is driven by the persisted billing-identity columns: sub-user, then
// tenant, otherwise the legacy user path (chargeUser) — keeping JWT/image-share
// tasks behaviourally identical.
func (s *Service) settleTaskCharge(task *store.ImageTask, usage billing.UsageInfo, requestID string) (float64, error) {
	switch {
	case task.SubUserID != "":
		return s.core.ChargeSubUserSync(task.SubUserID, task.SubUserKeyID, task.TenantID, task.Model, requestID, usage)
	case task.TenantID != "":
		return s.core.ChargeTenantSync(task.TenantID, task.TenantKeyID, task.Model, requestID, usage)
	default:
		return s.chargeUser(task.UserID, task.Model, requestID, usage, task.APIKeyID)
	}
}

// StartWorkers launches n background workers to process image tasks.
func (s *Service) StartWorkers(n int) {
	s.taskCh = make(chan struct{}, 100)
	s.stopCh = make(chan struct{})
	for i := 0; i < n; i++ {
		s.wg.Add(1)
		go s.taskWorker()
	}
	recovered, err := s.store.RecoverStaleTasks(context.Background(), 10*time.Minute)
	if err != nil {
		slog.Error("failed to recover stale image tasks", "error", err)
	} else if recovered > 0 {
		slog.Info("recovered stale image tasks", "count", recovered)
	}
}

func (s *Service) taskWorker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stopCh:
			return
		case <-s.taskCh:
		case <-time.After(500 * time.Millisecond):
		}
		s.processNextTask()
	}
}

func (s *Service) NotifyNewTask() {
	select {
	case s.taskCh <- struct{}{}:
	default:
	}
}

func (s *Service) StopWorkers() {
	close(s.stopCh)
	s.wg.Wait()
}

// SubmitGenerateTask creates a pending generate task and notifies workers.
// imageShareKeyID, when non-empty, attributes the task to an image-share key so the
// worker decrements its quota on completion.
func (s *Service) SubmitGenerateTask(ctx context.Context, userID string, model, prompt, size string, n int, params map[string]any, imageShareKeyID string) (*store.ImageTask, error) {
	if s.tosClient == nil {
		return nil, fmt.Errorf("image generation service unavailable: storage not configured")
	}
	if _, _, err := s.ParseSize(size); err != nil {
		return nil, err
	}
	// Image-share tasks stay on the owner's personal ledger; regular console
	// tasks from a tenant member route and bill through the tenant.
	var memberTenantID string
	if imageShareKeyID == "" {
		memberTenantID = s.core.MemberTenantIDForUser(userID)
	}
	if err := s.checkBalance(userID, memberTenantID, model); err != nil {
		return nil, fmt.Errorf("insufficient balance: %w", err)
	}
	task := &store.ImageTask{
		UserID:     userID,
		TenantID:   memberTenantID,
		Type:       "generate",
		Model:      model,
		Prompt:     prompt,
		Size:       size,
		ImageCount: n,
		Params:     params,
	}
	if imageShareKeyID != "" {
		v := imageShareKeyID
		task.ImageShareKeyID = &v
	}
	task, err := s.store.CreateImageTask(ctx, task)
	if err != nil {
		return nil, err
	}
	s.NotifyNewTask()
	return task, nil
}

// SubmitEditTask creates a pending edit task and notifies workers.
// imageShareKeyID, when non-empty, attributes the task to an image-share key.
func (s *Service) SubmitEditTask(ctx context.Context, userID string, model, prompt, size string, n int, images [][]byte, mask []byte, params map[string]any, imageShareKeyID string) (*store.ImageTask, error) {
	if s.tosClient == nil {
		return nil, fmt.Errorf("image generation service unavailable: storage not configured")
	}
	if _, _, err := s.ParseSize(size); err != nil {
		return nil, err
	}
	var memberTenantID string
	if imageShareKeyID == "" {
		memberTenantID = s.core.MemberTenantIDForUser(userID)
	}
	if err := s.checkBalance(userID, memberTenantID, model); err != nil {
		return nil, fmt.Errorf("insufficient balance: %w", err)
	}
	var b64Images []string
	for _, img := range images {
		b64Images = append(b64Images, base64.StdEncoding.EncodeToString(img))
	}
	task := &store.ImageTask{
		UserID:      userID,
		TenantID:    memberTenantID,
		Type:        "edit",
		Model:       model,
		Prompt:      prompt,
		Size:        size,
		ImageCount:  n,
		Params:      params,
		InputImages: b64Images,
		InputMask:   mask,
	}
	if imageShareKeyID != "" {
		v := imageShareKeyID
		task.ImageShareKeyID = &v
	}
	task, err := s.store.CreateImageTask(ctx, task)
	if err != nil {
		return nil, err
	}
	s.NotifyNewTask()
	return task, nil
}

// DeleteTask removes a task owned by userID, including its TOS objects (result_urls).
// input_images are stored as base64 inside the row, so they vanish with the row deletion.
// imageShareKeyID, when non-empty, additionally requires the task to belong to that share key.
func (s *Service) DeleteTask(ctx context.Context, userID string, taskID int) error {
	return s.deleteTask(ctx, userID, "", taskID)
}

// DeleteTaskAsImageShare removes a task only if it belongs to the given share key.
func (s *Service) DeleteTaskAsImageShare(ctx context.Context, imageShareKeyID string, taskID int) error {
	return s.deleteTask(ctx, "", imageShareKeyID, taskID)
}

func (s *Service) deleteTask(ctx context.Context, userID, imageShareKeyID string, taskID int) error {
	task, err := s.store.GetImageTask(ctx, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrTaskNotFound
		}
		return err
	}
	if imageShareKeyID != "" {
		if task.ImageShareKeyID == nil || *task.ImageShareKeyID != imageShareKeyID {
			return ErrTaskForbidden
		}
	} else {
		// Regular user path: must own the task and the task must NOT be an image-share task.
		if task.UserID != userID || task.ImageShareKeyID != nil {
			return ErrTaskForbidden
		}
	}

	s.deleteResultURLObjects(ctx, task.ResultURLs)

	if err := s.store.DeleteImageTask(ctx, taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return err
	}
	return nil
}

// DeleteTaskPublic removes a task submitted via the API-key endpoints
// (/v1/images/tasks). Ownership is verified against the caller's billing
// identity (user / tenant / sub-user) mirroring PublicTaskHandler.GetTask.
// Tasks in the "processing" state are refused with ErrTaskProcessing so the
// detached worker never settles a charge against a deleted row.
func (s *Service) DeleteTaskPublic(ctx context.Context, auth *proxy.AuthResult, taskID int) error {
	task, err := s.store.GetImageTask(ctx, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrTaskNotFound
		}
		return err
	}

	if !publicTaskOwnedBy(task, auth) {
		return ErrTaskForbidden
	}
	if task.Status == "processing" {
		return ErrTaskProcessing
	}

	s.deleteResultURLObjects(ctx, task.ResultURLs)

	if err := s.store.DeleteImageTask(ctx, taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return err
	}
	return nil
}

// If the array becomes empty, the entire task row is deleted as well.
// Returns the number of remaining images (>=0) and an error.
func (s *Service) DeleteResultImage(ctx context.Context, userID string, taskID int, url string) (int, error) {
	return s.deleteResultImage(ctx, userID, "", taskID, url)
}

// DeleteResultImageAsImageShare is the image-share counterpart of DeleteResultImage.
func (s *Service) DeleteResultImageAsImageShare(ctx context.Context, imageShareKeyID string, taskID int, url string) (int, error) {
	return s.deleteResultImage(ctx, "", imageShareKeyID, taskID, url)
}

func (s *Service) deleteResultImage(ctx context.Context, userID, imageShareKeyID string, taskID int, url string) (int, error) {
	task, err := s.store.GetImageTask(ctx, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return 0, ErrTaskNotFound
		}
		return 0, err
	}
	if imageShareKeyID != "" {
		if task.ImageShareKeyID == nil || *task.ImageShareKeyID != imageShareKeyID {
			return 0, ErrTaskForbidden
		}
	} else {
		if task.UserID != userID || task.ImageShareKeyID != nil {
			return 0, ErrTaskForbidden
		}
	}

	remaining := make([]string, 0, len(task.ResultURLs))
	found := false
	for _, u := range task.ResultURLs {
		if !found && u == url {
			found = true
			continue
		}
		remaining = append(remaining, u)
	}
	if !found {
		return 0, ErrImageNotInTask
	}

	s.deleteResultURLObjects(ctx, []string{url})

	if len(remaining) == 0 {
		if err := s.store.DeleteImageTask(ctx, taskID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		return 0, nil
	}
	if err := s.store.UpdateTaskResultURLs(ctx, taskID, remaining); err != nil {
		return 0, err
	}
	return len(remaining), nil
}

// deleteResultURLObjects best-effort deletes TOS objects for the given URLs.
// Failures are logged but do not abort callers (idempotency over strict consistency).
func (s *Service) deleteResultURLObjects(ctx context.Context, urls []string) {
	if s.tosClient == nil {
		return
	}
	for _, u := range urls {
		key := s.tosClient.KeyFromURL(u)
		if key == "" {
			continue
		}
		if err := s.tosClient.DeleteObject(ctx, key); err != nil {
			slog.Warn("delete TOS object failed", "key", key, "error", err)
		}
	}
}

func (s *Service) processNextTask() {
	ctx := context.Background()
	task, err := s.store.ClaimPendingTask(ctx)
	if err != nil {
		slog.Error("failed to claim image task", "error", err)
		return
	}
	if task == nil {
		return
	}
	slog.Info("processing image task", "id", task.ID, "type", task.Type, "n", task.ImageCount)

	var (
		successCount int
		cost         float64
	)

	switch task.Type {
	case "edit":
		successCount, cost, err = s.executeEditTask(ctx, task)
	default:
		successCount, cost, err = s.executeGenerateTask(ctx, task)
	}

	// Image-share quota was reserved at submission time (validateAndPrepareShare).
	// Refund the entire reservation when the task fails or produces zero images;
	// keep the reservation when at least one image was generated (partial success
	// counts as full task occupation per product decision).
	shouldRefund := false
	if err != nil {
		slog.Error("image task failed", "id", task.ID, "error", err)
		s.store.FailTask(ctx, task.ID, err.Error())
		shouldRefund = true
	} else if successCount == 0 {
		slog.Warn("image task produced no images", "id", task.ID)
		s.store.FailTask(ctx, task.ID, "all image generation attempts failed")
		shouldRefund = true
	} else {
		if ferr := s.store.FinalizeTask(ctx, task.ID, cost); ferr != nil {
			slog.Error("failed to finalize image task", "id", task.ID, "error", ferr)
		}
	}

	if shouldRefund && task.ImageShareKeyID != nil && *task.ImageShareKeyID != "" && s.imageShareStore != nil {
		if rerr := s.imageShareStore.RefundUsed(*task.ImageShareKeyID, task.ImageCount); rerr != nil {
			slog.Error("imageshare refund failed", "task", task.ID, "key", *task.ImageShareKeyID, "n", task.ImageCount, "error", rerr)
		}
	}
	slog.Info("image task done", "id", task.ID, "success", successCount, "total", task.ImageCount, "cost", cost)
}

// concurrencyLimit returns the per-task fan-out cap. Capped at 4 to avoid bursting
// past upstream provider per-key concurrency limits even when N is large.
func concurrencyLimit(n int) int {
	if n <= 0 {
		return 1
	}
	if n > 4 {
		return 4
	}
	return n
}

// retryBackoff returns a short exponential backoff for upstream retries:
// attempt=1 -> 200ms, attempt=2 -> 600ms (~3x). Image generation is expensive
// per call, so we keep retries quick to bound worst-case latency for failed
// images (4 images * 3 retries was up to ~12s of pure sleep before).
func retryBackoff(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 200 * time.Millisecond
	case 2:
		return 600 * time.Millisecond
	default:
		return 1 * time.Second
	}
}

// upstreamCallTimeout is the hard cap for a single upstream image call. The
// shared HTTP client's ResponseHeaderTimeout is 30m for chat streaming, so we
// must shorten it per-call here for image endpoints. gpt-image-2 edits via
// relay providers can take 5-7 minutes on the tail, so 8m gives headroom
// without letting genuinely stuck calls hang forever.
const upstreamCallTimeout = 8 * time.Minute

// resolveImageSize picks the billing tier label.
// Gemini image models carry the real resolution in params.image_resolution
// ("512" / "1K" / "2K" / "4K") while task.Size is pinned to 1024x1024 by the
// frontend. Prefer the explicit param when present so 4K bills as 4K.
func resolveImageSize(params map[string]any, taskSize string, tos *storage.TOSClient) string {
	if v, ok := params["image_resolution"]; ok {
		if sv, ok := v.(string); ok && sv != "" {
			return sv
		}
	}
	width, height, err := parseSizeWH(taskSize)
	if err != nil {
		return "1K"
	}
	return tos.DetermineImageSize(width, height)
}

// parseSizeWH parses "WxH" without the validation rules in ParseSize.
func parseSizeWH(size string) (int, int, error) {
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid size")
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return w, h, nil
}

func (s *Service) executeGenerateTask(ctx context.Context, task *store.ImageTask) (int, float64, error) {
	imageSize := resolveImageSize(task.Params, task.Size, s.tosClient)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrencyLimit(task.ImageCount))

	var successCount int64
	for i := 0; i < task.ImageCount; i++ {
		idx := i
		g.Go(func() error {
			if err := s.acquireUpstreamSlot(gctx); err != nil {
				return nil
			}
			defer s.releaseUpstreamSlot()

			singleReq := &GenerateRequest{
				Model:    task.Model,
				Prompt:   task.Prompt,
				Size:     task.Size,
				N:        1,
				Params:   task.Params,
				TenantID: task.TenantID,
			}

			var (
				urls    []string
				lastErr error
			)
			for attempt := 0; attempt < 2; attempt++ {
				if attempt > 0 {
					select {
					case <-time.After(retryBackoff(attempt)):
					case <-gctx.Done():
						return nil
					}
				}
				callCtx, cancel := context.WithTimeout(gctx, upstreamCallTimeout)
				urls, lastErr = s.callUpstreamAPI(callCtx, singleReq, task.UserID)
				cancel()
				if lastErr == nil {
					break
				}
				slog.Warn("image generation attempt failed",
					"task", task.ID, "image", idx+1, "attempt", attempt+1, "error", lastErr)
			}
			if lastErr != nil || len(urls) == 0 {
				slog.Warn("image generation gave up", "task", task.ID, "image", idx+1, "error", lastErr)
				return nil
			}

			finalURL := urls[0]
			if !s.tosClient.IsTOSURL(finalURL) {
				if data, err := s.downloadImage(gctx, finalURL); err == nil {
					if tosURL, upErr := s.tosClient.UploadImage(gctx, data, task.UserID); upErr == nil {
						finalURL = tosURL
					} else {
						slog.Warn("upload to TOS failed, fallback to upstream URL", "task", task.ID, "error", upErr)
					}
				} else {
					slog.Warn("download upstream image failed, fallback to upstream URL", "task", task.ID, "error", err)
				}
			}

			if err := s.store.AppendResultURL(gctx, task.ID, finalURL); err != nil {
				slog.Warn("append result url failed", "task", task.ID, "error", err)
				return nil
			}
			atomic.AddInt64(&successCount, 1)
			return nil
		})
	}
	_ = g.Wait()

	final := int(atomic.LoadInt64(&successCount))
	if final == 0 {
		return 0, 0, fmt.Errorf("all image generation attempts failed")
	}

	usage := billing.UsageInfo{ImageSize: imageSize, ImageCount: final}
	requestID := fmt.Sprintf("img-%d-%d", task.ID, time.Now().UnixNano())
	cost, err := s.settleTaskCharge(task, usage, requestID)
	if err != nil {
		return final, 0, fmt.Errorf("billing failed: %w", err)
	}
	return final, cost, nil
}

func (s *Service) executeEditTask(ctx context.Context, task *store.ImageTask) (int, float64, error) {
	imageSize := resolveImageSize(task.Params, task.Size, s.tosClient)

	var images [][]byte
	for _, b64 := range task.InputImages {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to decode input image: %w", err)
		}
		images = append(images, data)
	}

	editReq := &EditRequest{
		Model:    task.Model,
		Prompt:   task.Prompt,
		Size:     task.Size,
		N:        1,
		Params:   task.Params,
		Images:   images,
		Mask:     task.InputMask,
		TenantID: task.TenantID,
	}

	multipartBody, contentType, err := s.buildEditMultipart(editReq)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to build edit request: %w", err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrencyLimit(task.ImageCount))

	var successCount int64
	for i := 0; i < task.ImageCount; i++ {
		idx := i
		g.Go(func() error {
			if err := s.acquireUpstreamSlot(gctx); err != nil {
				return nil
			}
			defer s.releaseUpstreamSlot()

			var (
				imgData [][]byte
				lastErr error
			)
			for attempt := 0; attempt < 2; attempt++ {
				if attempt > 0 {
					select {
					case <-time.After(retryBackoff(attempt)):
					case <-gctx.Done():
						return nil
					}
				}
				callCtx, cancel := context.WithTimeout(gctx, upstreamCallTimeout)
				imgData, lastErr = s.callUpstreamEditAPI(callCtx, editReq, multipartBody, contentType)
				cancel()
				if lastErr == nil {
					break
				}
				slog.Warn("image edit attempt failed",
					"task", task.ID, "image", idx+1, "attempt", attempt+1, "error", lastErr)
			}
			if lastErr != nil || len(imgData) == 0 {
				slog.Warn("image edit gave up", "task", task.ID, "image", idx+1, "error", lastErr)
				return nil
			}

			tosURL, upErr := s.tosClient.UploadImage(gctx, imgData[0], task.UserID)
			if upErr != nil {
				slog.Warn("upload edited image to TOS failed", "task", task.ID, "error", upErr)
				return nil
			}
			if err := s.store.AppendResultURL(gctx, task.ID, tosURL); err != nil {
				slog.Warn("append edit result url failed", "task", task.ID, "error", err)
				return nil
			}
			atomic.AddInt64(&successCount, 1)
			return nil
		})
	}
	_ = g.Wait()

	final := int(atomic.LoadInt64(&successCount))
	if final == 0 {
		return 0, 0, fmt.Errorf("all image edit attempts failed")
	}

	usage := billing.UsageInfo{ImageSize: imageSize, ImageCount: final}
	requestID := fmt.Sprintf("img-edit-%d-%d", task.ID, time.Now().UnixNano())
	cost, err := s.settleTaskCharge(task, usage, requestID)
	if err != nil {
		return final, 0, fmt.Errorf("billing failed: %w", err)
	}
	return final, cost, nil
}
