package image

import (
	"encoding/json"
	"fmt"
	"strings"
)

// extractUpstreamMessage extracts a user-readable message from a service-layer upstream error.
// service.go wraps upstream errors as: "upstream returned status NNN: {json body}"
// This function parses the JSON body to extract error.message.
// If the error originates from an upstream call but the body is not parseable,
// only the status code is returned — never the raw body or any upstream URL.
func extractUpstreamMessage(err error) string {
	s := err.Error()

	// Only handle errors explicitly tagged by the service layer as upstream errors.
	if !strings.HasPrefix(s, "upstream returned status ") {
		return s
	}

	// Parse status code.
	var code int
	fmt.Sscanf(s, "upstream returned status %d:", &code)

	// Try to extract error.message from JSON body.
	if idx := strings.Index(s, ": {"); idx != -1 {
		var body struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(s[idx+2:]), &body) == nil && body.Error.Message != "" {
			return body.Error.Message
		}
	}

	// JSON parsing failed — return only the status code, not the raw body.
	if code > 0 {
		return fmt.Sprintf("上游服务返回错误（状态码 %d）", code)
	}
	return "上游服务返回了错误"
}
