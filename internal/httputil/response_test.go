package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"valid bearer", "Bearer sk-abc123", "sk-abc123"},
		{"no prefix", "sk-abc123", ""},
		{"empty", "", ""},
		{"basic auth", "Basic dXNlcjpwYXNz", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			got := ExtractBearerToken(req)
			if got != tt.want {
				t.Fatalf("ExtractBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsValidToken(t *testing.T) {
	tokens := []string{"token-a", "token-b"}
	if !IsValidToken("token-a", tokens) {
		t.Fatal("expected token-a to be valid")
	}
	if IsValidToken("token-c", tokens) {
		t.Fatal("expected token-c to be invalid")
	}
	if IsValidToken("", tokens) {
		t.Fatal("expected empty token to be invalid")
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusUnauthorized, "bad token", "auth_error", "invalid_api_key")

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected 'error' key in response")
	}
	if errObj["message"] != "bad token" {
		t.Fatalf("expected message 'bad token', got %v", errObj["message"])
	}
	if errObj["code"] != "invalid_api_key" {
		t.Fatalf("expected code 'invalid_api_key', got %v", errObj["code"])
	}
}
