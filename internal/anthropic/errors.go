package anthropic

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// anthropicError is the Anthropic API error envelope.
type anthropicError struct {
	Type  string              `json:"type"`
	Error anthropicErrorBody  `json:"error"`
}

type anthropicErrorBody struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// WriteError writes an error response in Anthropic API format.
func WriteError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(anthropicError{
		Type: "error",
		Error: anthropicErrorBody{
			Type:    errType,
			Message: message,
		},
	}); err != nil {
		slog.Error("failed to write anthropic error response", "error", err)
	}
}

// httpStatusToErrorType maps HTTP status codes to Anthropic error type strings.
func httpStatusToErrorType(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request_error"
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusForbidden:
		return "permission_error"
	case http.StatusNotFound:
		return "not_found_error"
	case http.StatusPaymentRequired:
		return "invalid_request_error"
	case http.StatusRequestEntityTooLarge:
		return "invalid_request_error"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	case http.StatusServiceUnavailable:
		return "overloaded_error"
	default:
		return "api_error"
	}
}
