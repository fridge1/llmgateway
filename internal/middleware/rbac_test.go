package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhulang/llm-gateway/internal/admin"
)

// TestRBACMatrix is the executable permission matrix: every role × route-group
// combination is asserted reachable or forbidden.
func TestRBACMatrix(t *testing.T) {
	cases := []struct {
		role    string
		path    string
		allowed bool
	}{
		// admin: everything
		{"admin", "/api/admin/tickets", true},
		{"admin", "/api/admin/refunds", true},
		{"admin", "/api/admin/alert/rules", true},
		{"admin", "/api/admin/pricing", true}, // unmatched prefix → admin only

		// support: tickets + users, nothing financial
		{"support", "/api/admin/tickets", true},
		{"support", "/api/admin/tickets/abc/reply", true},
		{"support", "/api/admin/users", true},
		{"support", "/api/admin/orders", false},
		{"support", "/api/admin/refunds", false},
		{"support", "/api/admin/invoice/requests", false},
		{"support", "/api/admin/pricing", false},
		{"support", "/api/admin/alert/rules", false},

		// finance: commerce only
		{"finance", "/api/admin/orders", true},
		{"finance", "/api/admin/orders/ORD123/refund", true},
		{"finance", "/api/admin/refunds", true},
		{"finance", "/api/admin/invoice/requests", true},
		{"finance", "/api/admin/subscription-orders", true},
		{"finance", "/api/admin/tickets", false},
		{"finance", "/api/admin/users", false},
		{"finance", "/api/admin/moderation/keywords", false},

		// ops: safety/observability only
		{"ops", "/api/admin/alert/rules", true},
		{"ops", "/api/admin/alert/events", true},
		{"ops", "/api/admin/moderation/settings", true},
		{"ops", "/api/admin/upstreams/test", true},
		{"ops", "/api/admin/orders", false},
		{"ops", "/api/admin/users", false},

		// plain user: nothing
		{"user", "/api/admin/tickets", false},
		{"user", "/api/admin/dashboard", false},

		// empty role (no JWT role claim): nothing
		{"", "/api/admin/users", false},
	}

	for _, tc := range cases {
		if got := RoleAllowed(tc.role, tc.path); got != tc.allowed {
			t.Errorf("RoleAllowed(%q, %q) = %v, want %v", tc.role, tc.path, got, tc.allowed)
		}
	}
}

func TestBackofficeAccessMiddleware(t *testing.T) {
	handler := BackofficeAccess()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := func(role, path string) int {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r = r.WithContext(context.WithValue(r.Context(), admin.CtxRoleKey, role))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}

	if code := req("support", "/api/admin/tickets"); code != http.StatusOK {
		t.Errorf("support on tickets: got %d", code)
	}
	if code := req("support", "/api/admin/refunds"); code != http.StatusForbidden {
		t.Errorf("support on refunds: got %d, want 403", code)
	}
	if code := req("admin", "/api/admin/anything-new"); code != http.StatusOK {
		t.Errorf("admin fallback: got %d", code)
	}
	if code := req("finance", "/api/admin/anything-new"); code != http.StatusForbidden {
		t.Errorf("finance on unmatched path must fail closed: got %d", code)
	}
}
