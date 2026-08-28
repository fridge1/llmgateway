package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/httputil"
)

type testRequest struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type testResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Latency string `json:"latency,omitempty"`
}

// HandleTestUpstream tests connectivity to an upstream provider.
func HandleTestUpstream(w http.ResponseWriter, r *http.Request) {
	var req testRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}
	if req.BaseURL == "" || req.APIKey == "" {
		httputil.WriteError(w, http.StatusBadRequest, "base_url and api_key required", "invalid_request_error", "missing_fields")
		return
	}

	// Send GET /models to the upstream to verify API key.
	url := strings.TrimRight(req.BaseURL, "/") + "/models"
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		writeTestResponse(w, false, fmt.Sprintf("创建请求失败: %v", err), "")
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	start := time.Now()
	resp, err := http.DefaultClient.Do(httpReq)
	latency := time.Since(start)

	if err != nil {
		writeTestResponse(w, false, fmt.Sprintf("连接失败: %v", err), "")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		writeTestResponse(w, true, "连接成功", latency.Round(time.Millisecond).String())
	} else if resp.StatusCode == http.StatusUnauthorized {
		writeTestResponse(w, false, "API Key 无效 (401 Unauthorized)", latency.Round(time.Millisecond).String())
	} else {
		writeTestResponse(w, false, fmt.Sprintf("上游返回 HTTP %d", resp.StatusCode), latency.Round(time.Millisecond).String())
	}
}

func writeTestResponse(w http.ResponseWriter, success bool, message, latency string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testResponse{Success: success, Message: message, Latency: latency})
}

type remoteModel struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
}

type listRemoteResponse struct {
	Success bool          `json:"success"`
	Message string        `json:"message,omitempty"`
	Models  []remoteModel `json:"models"`
}

// HandleListRemoteModels fetches the model list from an upstream provider.
func HandleListRemoteModels(w http.ResponseWriter, r *http.Request) {
	var req testRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}
	if req.BaseURL == "" || req.APIKey == "" {
		httputil.WriteError(w, http.StatusBadRequest, "base_url and api_key required", "invalid_request_error", "missing_fields")
		return
	}

	url := strings.TrimRight(req.BaseURL, "/") + "/models"
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listRemoteResponse{Success: false, Message: fmt.Sprintf("创建请求失败: %v", err)})
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listRemoteResponse{Success: false, Message: fmt.Sprintf("连接失败: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listRemoteResponse{Success: false, Message: "API Key 无效 (401 Unauthorized)"})
		return
	}
	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listRemoteResponse{Success: false, Message: fmt.Sprintf("上游返回 HTTP %d", resp.StatusCode)})
		return
	}

	// Parse OpenAI-compatible /models response: {data: [{id, owned_by}]}
	var body struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listRemoteResponse{Success: false, Message: fmt.Sprintf("解析响应失败: %v", err)})
		return
	}

	models := make([]remoteModel, len(body.Data))
	for i, m := range body.Data {
		models[i] = remoteModel{ID: m.ID, OwnedBy: m.OwnedBy}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listRemoteResponse{Success: true, Models: models})
}
