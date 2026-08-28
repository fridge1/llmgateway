package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Fatal("expected request ID in context, got empty string")
		}
		if len(id) < 5 {
			t.Fatalf("expected request ID with prefix, got %q", id)
		}
		if id[:4] != "req_" {
			t.Fatalf("expected request ID to start with 'req_', got %q", id)
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Request-Id"); got == "" {
		t.Fatal("expected X-Request-Id response header, got empty")
	}
}

func TestRequestID_UsesExistingHeader(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != "my-custom-id" {
			t.Fatalf("expected 'my-custom-id', got %q", id)
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", "my-custom-id")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Request-Id"); got != "my-custom-id" {
		t.Fatalf("expected 'my-custom-id' in response header, got %q", got)
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	id := GetRequestID(req.Context())
	if id != "" {
		t.Fatalf("expected empty string from context without request ID, got %q", id)
	}
}
