package ppt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/proxy"
	"github.com/zhulang/llm-gateway/internal/storage"
	"github.com/zhulang/llm-gateway/internal/store"
	"golang.org/x/sync/errgroup"
)

const defaultImageModel = "gpt-image-2"

// Service orchestrates PPT generation via a worker pool.
type Service struct {
	store          *store.PgStore
	billingService *billing.BillingService
	core           *proxy.Core
	tosClient      *storage.TOSClient
	taskCh         chan struct{}
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

// NewService creates a new PPT service.
func NewService(s *store.PgStore, bs *billing.BillingService, core *proxy.Core, tos *storage.TOSClient) *Service {
	return &Service{
		store:          s,
		billingService: bs,
		core:           core,
		tosClient:      tos,
	}
}

// StartWorkers launches background workers that process PPT tasks.
func (s *Service) StartWorkers(count int) {
	s.taskCh = make(chan struct{}, 100)
	s.stopCh = make(chan struct{})

	if err := s.store.RecoverStalePptTasks(context.Background()); err != nil {
		slog.Error("failed to recover stale ppt tasks", "error", err)
	}

	for i := 0; i < count; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	slog.Info("ppt workers started", "count", count)
}

// Stop gracefully shuts down all workers.
func (s *Service) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

func (s *Service) worker(id int) {
	defer s.wg.Done()
	for {
		select {
		case <-s.stopCh:
			return
		case <-s.taskCh:
			s.processNextTask(id)
		case <-time.After(5 * time.Second):
			s.processNextTask(id)
		}
	}
}

func (s *Service) processNextTask(workerID int) {
	ctx := context.Background()
	task, err := s.store.ClaimPendingPptTask(ctx)
	if err != nil || task == nil {
		return
	}

	slog.Info("ppt worker processing task", "worker", workerID, "task_id", task.ID, "phase", task.Phase)
	totalTokens := 0
	promptTokens := 0
	completionTokens := 0

	accumulateTokens := func(r *AgentResult) {
		if r != nil {
			totalTokens += r.TotalTokens
			promptTokens += r.PromptTokens
			completionTokens += r.CompletionTokens
		}
	}

	// If resuming from outline confirmation, skip to Agent 3
	if task.Phase == "info_architect" && len(task.StoryArc) > 0 {
		totalTokens = task.TotalTokens
		var arc StoryArc
		if err := json.Unmarshal(task.StoryArc, &arc); err != nil {
			s.store.FailPptTask(ctx, task.ID, "failed to parse stored story arc: "+err.Error())
			return
		}
		var brief BriefDocument
		if len(task.BriefDocument) > 0 {
			json.Unmarshal(task.BriefDocument, &brief)
		}

		blueprints, result3, err := RunInfoArchitect(ctx, s.core, toPptTask(task), &arc)
		accumulateTokens(result3)
		if err != nil {
			s.store.FailPptTask(ctx, task.ID, "Info Architect: "+err.Error())
			return
		}

		presentation := blueprintsToPresentation(blueprints, &brief, task.Theme)

		// Agent 4: Visual Designer + Image Generation (resumed path)
		if task.GenerateImages && s.tosClient != nil {
			s.generateAndAttachImages(ctx, task, presentation, accumulateTokens)
		}

		presJSON, _ := json.Marshal(presentation)

		requestID := fmt.Sprintf("ppt-%d-%d", task.ID, time.Now().UnixNano())
		usage := billing.UsageInfo{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
		}
		cost, billingErr := s.billingService.ChargeAndReturnCost(task.UserID, task.Model, requestID, usage, "")
		if billingErr != nil {
			slog.Warn("ppt billing failed", "task_id", task.ID, "error", billingErr)
		}

		s.store.CompletePptTask(ctx, task.ID, presJSON, totalTokens, cost)
		slog.Info("ppt task completed (resumed)", "task_id", task.ID, "tokens", totalTokens, "cost", cost)
		return
	}

	// Agent 1: Brief Analyst
	s.store.UpdatePptTaskPhase(ctx, task.ID, "brief_analyst", "", nil)
	brief, result1, err := RunBriefAnalyst(ctx, s.core, toPptTask(task))
	accumulateTokens(result1)
	if err != nil {
		s.store.FailPptTask(ctx, task.ID, "Brief Analyst: "+err.Error())
		return
	}
	briefJSON, err := json.Marshal(brief)
	if err != nil {
		s.store.FailPptTask(ctx, task.ID, "internal: marshal brief failed")
		return
	}
	s.store.UpdatePptTaskPhase(ctx, task.ID, "content_strategist", "brief_document", briefJSON)

	// Agent 2: Content Strategist
	arc, result2, err := RunContentStrategist(ctx, s.core, toPptTask(task), brief)
	accumulateTokens(result2)
	if err != nil {
		s.store.FailPptTask(ctx, task.ID, "Content Strategist: "+err.Error())
		return
	}
	arcJSON, err := json.Marshal(arc)
	if err != nil {
		s.store.FailPptTask(ctx, task.ID, "internal: marshal story arc failed")
		return
	}

	// If outline_only, pause here for user review
	if task.OutlineOnly {
		s.store.PausePptTaskForOutline(ctx, task.ID, arcJSON, totalTokens)
		slog.Info("ppt task paused for outline review", "task_id", task.ID)
		return
	}

	s.store.UpdatePptTaskPhase(ctx, task.ID, "info_architect", "story_arc", arcJSON)

	// Agent 3: Info Architect
	blueprints, result3, err := RunInfoArchitect(ctx, s.core, toPptTask(task), arc)
	accumulateTokens(result3)
	if err != nil {
		s.store.FailPptTask(ctx, task.ID, "Info Architect: "+err.Error())
		return
	}

	// Convert blueprints to presentation JSON
	presentation := blueprintsToPresentation(blueprints, brief, task.Theme)

	// Agent 4: Visual Designer + Image Generation
	if task.GenerateImages && s.tosClient != nil {
		s.generateAndAttachImages(ctx, task, presentation, accumulateTokens)
	}

	presJSON, _ := json.Marshal(presentation)

	// Settle billing
	requestID := fmt.Sprintf("ppt-%d-%d", task.ID, time.Now().UnixNano())
	usage := billing.UsageInfo{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
	}
	cost, billingErr := s.billingService.ChargeAndReturnCost(task.UserID, task.Model, requestID, usage, "")
	if billingErr != nil {
		slog.Warn("ppt billing failed", "task_id", task.ID, "error", billingErr)
	}

	s.store.CompletePptTask(ctx, task.ID, presJSON, totalTokens, cost)
	slog.Info("ppt task completed", "task_id", task.ID, "tokens", totalTokens, "cost", cost)
}

// toPptTask converts a store.PptTask to a ppt.PptTask.
func toPptTask(st *store.PptTask) *PptTask {
	return &PptTask{
		ID:             st.ID,
		UserID:         st.UserID,
		Topic:          st.Topic,
		SlideCount:     st.SlideCount,
		Language:       st.Language,
		Theme:          st.Theme,
		Audience:       st.Audience,
		Tone:           st.Tone,
		Purpose:        st.Purpose,
		Model:          st.Model,
		OutlineOnly:    st.OutlineOnly,
		GenerateImages: st.GenerateImages,
		ContextText:    st.ContextText,
	}
}

// generateAndAttachImages runs Agent 4 (Visual Designer) then generates and uploads images.
func (s *Service) generateAndAttachImages(ctx context.Context, task *store.PptTask, presentation map[string]interface{}, accumulateTokens func(*AgentResult)) {
	s.store.UpdatePptTaskPhase(ctx, task.ID, "visual_designer", "", nil)

	plan, result4, err := RunVisualDesigner(ctx, s.core, task.Model, presentation)
	accumulateTokens(result4)
	if err != nil {
		slog.Warn("visual designer failed, skipping images", "task_id", task.ID, "error", err)
		return
	}

	if len(plan.Images) == 0 {
		slog.Info("visual designer returned no images", "task_id", task.ID)
		return
	}

	s.store.UpdatePptTaskPhase(ctx, task.ID, "image_generation", "", nil)

	slides, _ := presentation["slides"].([]interface{})

	// Defensive filter: skip disallowed layouts, cap at 40% of slides (min 1).
	disallowed := map[string]bool{"title": true, "section": true, "closing": true}
	filtered := plan.Images[:0]
	for _, img := range plan.Images {
		if img.SlideIndex < 0 || img.SlideIndex >= len(slides) {
			continue
		}
		sm, ok := slides[img.SlideIndex].(map[string]interface{})
		if !ok {
			continue
		}
		layout, _ := sm["layout"].(string)
		if disallowed[layout] {
			continue
		}
		filtered = append(filtered, img)
	}
	maxImages := len(slides) * 40 / 100
	if maxImages < 1 {
		maxImages = 1
	}
	if len(filtered) > maxImages {
		filtered = filtered[:maxImages]
	}
	plan.Images = filtered

	var totalImageCost float64
	generated := 0

	// Parallelize image generation with concurrency cap = 2 to keep total task time
	// bounded when the deck has multiple images. Per-image errors are logged and skipped;
	// they never abort the deck.
	type result struct {
		idx     int
		spec    ImageSpec
		url     string
		cost    float64
		err     error
	}
	results := make([]result, len(plan.Images))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(2)
	for i := range plan.Images {
		i := i
		spec := plan.Images[i]
		g.Go(func() error {
			url, cost, err := s.generateSingleImage(gctx, task, spec)
			results[i] = result{idx: i, spec: spec, url: url, cost: cost, err: err}
			return nil
		})
	}
	_ = g.Wait()

	for _, r := range results {
		if r.err != nil {
			slog.Warn("image generation failed", "task_id", task.ID, "slide", r.spec.SlideIndex, "error", r.err)
			continue
		}
		if sm, ok := slides[r.spec.SlideIndex].(map[string]interface{}); ok {
			sm["imageUrl"] = r.url
			if r.spec.Style != "" {
				sm["imageStyle"] = r.spec.Style
			}
			if r.spec.ImageSlot != "" {
				sm["imageSlot"] = r.spec.ImageSlot
			}
		}
		totalImageCost += r.cost
		generated++
	}

	if totalImageCost > 0 {
		if err := s.store.UpdatePptTaskImageCost(ctx, task.ID, totalImageCost); err != nil {
			slog.Warn("failed to save image cost", "task_id", task.ID, "error", err)
		}
	}

	slog.Info("ppt images generated", "task_id", task.ID, "planned", len(plan.Images), "generated", generated, "image_cost", totalImageCost)
}

// generateSingleImage calls the upstream image API, downloads the result, uploads to TOS, and returns the permanent URL.
func (s *Service) generateSingleImage(ctx context.Context, task *store.PptTask, spec ImageSpec) (string, float64, error) {
	// Build OpenAI-compatible image generation request
	upstreamReq := map[string]interface{}{
		"model":  defaultImageModel,
		"prompt": stylePrefixPrompt(spec),
		"size":   "1024x1024",
		"n":      1,
	}
	reqBody, err := json.Marshal(upstreamReq)
	if err != nil {
		return "", 0, fmt.Errorf("marshal request: %w", err)
	}

	// Get upstream for image model
	rt := s.core.Router.Load()
	upstreams, _, found := rt.GetUpstreams(defaultImageModel)
	if !found {
		return "", 0, fmt.Errorf("image model %q not found in router", defaultImageModel)
	}

	var imageURL string
	for _, upstream := range upstreams {
		if !upstream.Breaker.AllowRequest() {
			continue
		}

		baseURL := strings.TrimRight(upstream.Config.BaseURL, "/")
		url := baseURL + "/v1/images/generations"

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
		if err != nil {
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if upstream.Config.APIKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+upstream.Config.APIKey)
		}

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			if circuit.IsUpstreamFailure(err) {
				upstream.Breaker.RecordFailure()
			}
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
			return "", 0, fmt.Errorf("upstream returned status %d: %s", resp.StatusCode, string(respBody))
		}

		upstream.Breaker.RecordSuccess()

		// Parse response — OpenAI format
		var imgResp struct {
			Data []struct {
				URL     string `json:"url"`
				B64JSON string `json:"b64_json"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &imgResp); err == nil && len(imgResp.Data) > 0 {
			if imgResp.Data[0].URL != "" {
				imageURL = imgResp.Data[0].URL
			} else if imgResp.Data[0].B64JSON != "" {
				// base64 image — decode and upload directly
				imgData, err := base64.StdEncoding.DecodeString(imgResp.Data[0].B64JSON)
				if err == nil {
					tosURL, err := s.tosClient.UploadImage(ctx, imgData, task.UserID)
					if err != nil {
						return "", 0, fmt.Errorf("TOS upload failed: %w", err)
					}
					// Charge for image
					imgRequestID := fmt.Sprintf("ppt-img-%d-%d", task.ID, time.Now().UnixNano())
					imgUsage := billing.UsageInfo{ImageSize: "1024x1024", ImageCount: 1}
					cost, _ := s.billingService.ChargeAndReturnCost(task.UserID, defaultImageModel, imgRequestID, imgUsage, "")
					return tosURL, cost, nil
				}
			}
		}

		// Fallback: try image_urls format
		if imageURL == "" {
			var altResp struct {
				ImageURLs []string `json:"image_urls"`
			}
			if err := json.Unmarshal(respBody, &altResp); err == nil && len(altResp.ImageURLs) > 0 {
				imageURL = altResp.ImageURLs[0]
			}
		}

		if imageURL != "" {
			break
		}
	}

	if imageURL == "" {
		return "", 0, fmt.Errorf("no image returned from upstream")
	}

	// Download the generated image and upload to TOS for permanent storage
	imgData, err := s.tosClient.DownloadImageFromURL(ctx, imageURL)
	if err != nil {
		return "", 0, fmt.Errorf("download image: %w", err)
	}

	tosURL, err := s.tosClient.UploadImage(ctx, imgData, task.UserID)
	if err != nil {
		return "", 0, fmt.Errorf("TOS upload: %w", err)
	}

	// Charge for image generation
	imgRequestID := fmt.Sprintf("ppt-img-%d-%d", task.ID, time.Now().UnixNano())
	imgUsage := billing.UsageInfo{ImageSize: "1024x1024", ImageCount: 1}
	cost, _ := s.billingService.ChargeAndReturnCost(task.UserID, defaultImageModel, imgRequestID, imgUsage, "")

	return tosURL, cost, nil
}

func blueprintsToPresentation(blueprints *SlideBlueprintSet, brief *BriefDocument, theme string) map[string]interface{} {
	slides := make([]map[string]interface{}, 0, len(blueprints.Blueprints))

	for _, bp := range blueprints.Blueprints {
		slide := map[string]interface{}{
			"layout": resolveLayout(bp),
		}
		if bp.LayoutRationale != "" {
			slide["layoutRationale"] = bp.LayoutRationale
		}

		for _, el := range bp.Elements {
			switch el.Type {
			case "heading":
				if _, ok := slide["title"]; !ok {
					slide["title"] = el.Content
				}
			case "subheading":
				if _, ok := slide["subtitle"]; !ok {
					slide["subtitle"] = el.Content
				}
			case "body_text":
				if el.Content != "" {
					slide["body"] = el.Content
				}
			case "bullet_list":
				if len(el.Items) > 0 {
					existing, _ := slide["bullets"].([]string)
					slide["bullets"] = append(existing, el.Items...)
				}
			case "stat_number":
				stat := map[string]interface{}{"value": el.Content}
				if el.Unit != "" {
					stat["unit"] = el.Unit
				}
				if el.Label != "" {
					stat["label"] = el.Label
				}
				existing, _ := slide["stats"].([]map[string]interface{})
				slide["stats"] = append(existing, stat)
			case "timeline_items":
				items := normalizeKeyedItems(el.StructuredItems, el.Items, []string{"time", "title", "description"})
				if len(items) > 0 {
					slide["timelineItems"] = items
				}
			case "icon_grid":
				items := normalizeKeyedItems(el.StructuredItems, el.Items, []string{"icon", "title", "description"})
				if len(items) > 0 {
					slide["iconGrid"] = items
				}
			case "comparison_table":
				comp := buildComparison(el)
				if comp != nil {
					slide["comparison"] = comp
				}
			case "quote":
				if el.Content != "" {
					q := map[string]interface{}{"text": el.Content}
					if el.Attribution != "" {
						q["attribution"] = el.Attribution
					}
					slide["quote"] = q
				}
			case "chart_data":
				chart := buildChartData(el)
				if chart != nil {
					slide["chartData"] = chart
				}
			}
		}

		if _, ok := slide["title"]; !ok {
			slide["title"] = ""
		}

		if bp.SpeakerNotes != "" {
			slide["speakerNotes"] = bp.SpeakerNotes
		}

		slides = append(slides, slide)
	}

	title := "Presentation"
	if len(brief.KeyMessages) > 0 {
		title = brief.KeyMessages[0]
	}

	return map[string]interface{}{
		"title":  title,
		"slides": slides,
	}
}

// resolveLayout picks a frontend layout. If an element strongly implies a specialized
// layout (icon_grid / timeline / stat / quote / comparison / chart), prefer that over
// the blueprint's content_type — the agent sometimes emits content_type="bullet_list"
// while actually producing an icon_grid element.
func resolveLayout(bp SlideBlueprint) string {
	layout := mapContentTypeToLayout(bp.ContentType)
	for _, el := range bp.Elements {
		switch el.Type {
		case "chart_data":
			if layout != "data_highlight" && layout != "chart" {
				return "chart"
			}
		case "timeline_items":
			return "timeline"
		case "comparison_table":
			return "comparison"
		case "quote":
			return "quote"
		case "icon_grid":
			if layout == "content" {
				return "icon_grid"
			}
		case "stat_number":
			if layout == "content" {
				return "data_highlight"
			}
		}
	}
	return layout
}

// normalizeKeyedItems prefers StructuredItems when present; otherwise falls back to
// parsing pipe-delimited strings into the given keys. Returns []map[string]string.
func normalizeKeyedItems(structured []map[string]string, strings []string, keys []string) []map[string]string {
	if len(structured) > 0 {
		out := make([]map[string]string, 0, len(structured))
		for _, m := range structured {
			if len(m) == 0 {
				continue
			}
			out = append(out, m)
		}
		if len(out) > 0 {
			return out
		}
	}
	out := make([]map[string]string, 0, len(strings))
	for _, s := range strings {
		parts := splitPipe(s)
		m := map[string]string{}
		for i, k := range keys {
			if i < len(parts) {
				m[k] = parts[i]
			}
		}
		if len(m) > 0 {
			out = append(out, m)
		}
	}
	return out
}

// splitPipe accepts both ASCII "|" and full-width "｜" as separators.
func splitPipe(s string) []string {
	s = strings.ReplaceAll(s, "｜", "|")
	parts := strings.Split(s, "|")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func buildComparison(el SlideElement) map[string]interface{} {
	headers := []string{"", ""}
	if len(el.Columns) >= 2 {
		headers[0] = el.Columns[0]
		headers[1] = el.Columns[1]
	}

	var rows [][2]string
	if len(el.StructuredItems) > 0 {
		for _, m := range el.StructuredItems {
			rows = append(rows, [2]string{m["left"], m["right"]})
		}
	} else {
		for i := 0; i+1 < len(el.Items); i += 2 {
			rows = append(rows, [2]string{el.Items[i], el.Items[i+1]})
		}
	}
	if headers[0] == "" && headers[1] == "" && len(rows) == 0 {
		return nil
	}
	return map[string]interface{}{"headers": headers, "rows": rows}
}

func buildChartData(el SlideElement) map[string]interface{} {
	if el.ChartType == "" || len(el.Labels) == 0 {
		return nil
	}
	var datasets []map[string]interface{}
	if len(el.Series) > 0 {
		for _, s := range el.Series {
			datasets = append(datasets, map[string]interface{}{
				"label":  s.Label,
				"values": s.Values,
			})
		}
	} else if len(el.Values) > 0 {
		datasets = append(datasets, map[string]interface{}{
			"label":  el.Content,
			"values": el.Values,
		})
	}
	if len(datasets) == 0 {
		return nil
	}
	return map[string]interface{}{
		"type":     el.ChartType,
		"labels":   el.Labels,
		"datasets": datasets,
	}
}

// stylePrefixPrompt prepends a concrete style directive and a hard "no text" guard to
// the Visual Designer prompt so the image model respects our style hint.
func stylePrefixPrompt(spec ImageSpec) string {
	prefix := ""
	switch spec.Style {
	case "photograph":
		prefix = "Professional editorial photograph, "
	case "illustration":
		prefix = "Modern flat vector illustration, "
	case "abstract":
		prefix = "Abstract geometric composition, "
	case "diagram":
		prefix = "Minimal infographic diagram, "
	}
	body := strings.TrimSpace(spec.Prompt)
	if body != "" && !strings.HasSuffix(body, ".") && !strings.HasSuffix(body, "!") && !strings.HasSuffix(body, "?") {
		body += "."
	}
	return prefix + body + " No text, no captions, no logos."
}

func mapContentTypeToLayout(contentType string) string {
	switch contentType {
	case "title_hero":
		return "title"
	case "section_break":
		return "section"
	case "closing_summary":
		return "closing"
	case "comparison_matrix":
		return "comparison"
	case "timeline":
		return "timeline"
	case "data_highlight":
		return "data_highlight"
	case "image_text":
		return "image_text"
	case "icon_grid":
		return "icon_grid"
	case "quote_highlight":
		return "quote"
	case "two_column":
		return "two_column"
	case "chart":
		return "chart"
	default:
		return "content"
	}
}

// SubmitTask creates a new PPT generation task.
func (s *Service) SubmitTask(ctx context.Context, userID, model, topic string, slideCount int, language, theme, audience, tone, purpose string, outlineOnly bool, generateImages bool, contextText string) (*store.PptTask, error) {
	if err := s.billingService.CheckBalance(userID, model); err != nil {
		return nil, fmt.Errorf("insufficient balance: %w", err)
	}

	task := &store.PptTask{
		UserID:         userID,
		Model:          model,
		Topic:          topic,
		SlideCount:     slideCount,
		Language:       language,
		Theme:          theme,
		Audience:       audience,
		Tone:           tone,
		Purpose:        purpose,
		OutlineOnly:    outlineOnly,
		GenerateImages: generateImages,
		ContextText:    contextText,
	}

	created, err := s.store.CreatePptTask(ctx, task)
	if err != nil {
		return nil, err
	}

	// Notify workers that a new task is available.
	select {
	case s.taskCh <- struct{}{}:
	default:
	}

	return created, nil
}

// ConfirmOutline confirms the outline and resumes generation for Agent 3.
func (s *Service) ConfirmOutline(ctx context.Context, taskID int, userID string) error {
	task, err := s.store.GetPptTask(ctx, taskID)
	if err != nil {
		return err
	}
	if task.UserID != userID {
		return fmt.Errorf("forbidden")
	}
	if task.Status != "outline_ready" {
		return fmt.Errorf("task not in outline_ready status")
	}

	if err := s.store.ConfirmPptTaskOutline(ctx, taskID); err != nil {
		return err
	}

	// Notify workers
	select {
	case s.taskCh <- struct{}{}:
	default:
	}

	return nil
}
