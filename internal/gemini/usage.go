package gemini

import (
	"encoding/json"

	"github.com/zhulang/llm-gateway/internal/billing"
)

// extractGeminiUsage extracts billing usage from a Gemini JSON response body.
// Gemini responses include an OpenAI-compatible "usage" field.
func extractGeminiUsage(body []byte) billing.UsageInfo {
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return billing.UsageInfo{}
	}
	return extractGeminiUsageFromMap(resp)
}

// extractGeminiSSEUsage extracts usage from a single SSE data payload.
func extractGeminiSSEUsage(data string) billing.UsageInfo {
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return billing.UsageInfo{}
	}
	return extractGeminiUsageFromMap(m)
}

func extractGeminiUsageFromMap(m map[string]any) billing.UsageInfo {
	// Try OpenAI-compatible "usage" field first (ppinfra includes this).
	if usageObj, ok := m["usage"]; ok {
		if usageMap, ok := usageObj.(map[string]any); ok {
			info := extractFromUsageField(usageMap)
			if info.PromptTokens > 0 || info.CompletionTokens > 0 {
				return info
			}
		}
	}

	// Fallback to Gemini-native "usageMetadata" (e.g. 4sapi only has this).
	if metaObj, ok := m["usageMetadata"]; ok {
		if metaMap, ok := metaObj.(map[string]any); ok {
			return extractFromUsageMetadata(metaMap)
		}
	}

	return billing.UsageInfo{}
}

func extractFromUsageField(usageMap map[string]any) billing.UsageInfo {
	var info billing.UsageInfo
	info.CacheTokensIncludedInPrompt = true

	if pt, ok := usageMap["prompt_tokens"].(float64); ok {
		info.PromptTokens = int(pt)
	}
	if ct, ok := usageMap["completion_tokens"].(float64); ok {
		info.CompletionTokens = int(ct)
	}

	if details, ok := usageMap["prompt_tokens_details"].(map[string]any); ok {
		if v, ok := details["cached_tokens"].(float64); ok {
			info.CacheReadTokens = int(v)
		}
		if v, ok := details["cache_read_input_tokens"].(float64); ok && int(v) > 0 {
			info.CacheReadTokens = int(v)
		}
		if v, ok := details["cache_creation_input_tokens"].(float64); ok {
			info.CacheCreationTokens = int(v)
		}
	}

	return info
}

func extractFromUsageMetadata(meta map[string]any) billing.UsageInfo {
	var info billing.UsageInfo

	if v, ok := meta["promptTokenCount"].(float64); ok {
		info.PromptTokens = int(v)
	}

	var candidates, thoughts int
	if v, ok := meta["candidatesTokenCount"].(float64); ok {
		candidates = int(v)
	}
	if v, ok := meta["thoughtsTokenCount"].(float64); ok {
		thoughts = int(v)
	}
	info.CompletionTokens = candidates + thoughts

	if v, ok := meta["cachedContentTokenCount"].(float64); ok {
		info.CacheReadTokens = int(v)
		info.CacheTokensIncludedInPrompt = true
	}

	return info
}
