package middleware

import (
	"net/http"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
)

// BackofficeRoles enumerates roles that may access some part of /api/admin.
// 'admin' has full access; the rest see route-group subsets per RBACMatrix.
var BackofficeRoles = []string{"admin", "support", "finance", "ops"}

// RBACRule maps an /api/admin path prefix to the roles allowed on it.
type RBACRule struct {
	Prefix string
	Roles  []string
}

// RBACMatrix is the single source of truth for back-office permissions.
// Longest matching prefix wins. Paths not matched by any rule fall back to
// admin-only, so forgetting to add a rule fails closed.
var RBACMatrix = []RBACRule{
	// Support: tickets + read-only user lookup.
	{Prefix: "/api/admin/tickets", Roles: []string{"admin", "support"}},
	{Prefix: "/api/admin/users", Roles: []string{"admin", "support"}},

	// Finance: orders, refunds, invoices, subscription commerce.
	{Prefix: "/api/admin/orders", Roles: []string{"admin", "finance"}},
	{Prefix: "/api/admin/refunds", Roles: []string{"admin", "finance"}},
	{Prefix: "/api/admin/invoice", Roles: []string{"admin", "finance"}},
	{Prefix: "/api/admin/subscription-orders", Roles: []string{"admin", "finance"}},

	// Ops: alerting, moderation, upstream health.
	{Prefix: "/api/admin/alert", Roles: []string{"admin", "ops"}},
	{Prefix: "/api/admin/moderation", Roles: []string{"admin", "ops"}},
	{Prefix: "/api/admin/upstreams", Roles: []string{"admin", "ops"}},
}

// rolesForPath resolves the allowed roles for an /api/admin path.
func rolesForPath(path string) []string {
	best := ""
	var roles []string
	for _, rule := range RBACMatrix {
		if strings.HasPrefix(path, rule.Prefix) && len(rule.Prefix) > len(best) {
			best = rule.Prefix
			roles = rule.Roles
		}
	}
	if best == "" {
		return []string{"admin"} // fail closed
	}
	return roles
}

// RoleAllowed reports whether role may access path (exported for tests and
// for the frontend nav-filtering endpoint if ever needed).
func RoleAllowed(role, path string) bool {
	for _, allowed := range rolesForPath(path) {
		if role == allowed {
			return true
		}
	}
	return false
}

// BackofficeAccess authorizes /api/admin requests against RBACMatrix.
// Must be used after JWTAuth. Replaces the old blanket RequireRole("admin").
func BackofficeAccess() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(admin.CtxRoleKey).(string)
			if !RoleAllowed(role, r.URL.Path) {
				httputil.WriteError(w, http.StatusForbidden, "insufficient permissions", "auth_error", "insufficient_role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
