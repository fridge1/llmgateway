package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// TenantRoleRequired extracts tenant_id from the URL path and verifies the
// current user has one of the allowed roles in that tenant.
// URL pattern expected: /api/tenants/{tenantID}/...
func TenantRoleRequired(s store.Store, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := extractTenantID(r.URL.Path)
			if tenantID == "" {
				httputil.WriteError(w, http.StatusBadRequest, "missing tenant id", "invalid_request_error", "missing_tenant_id")
				return
			}

			userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
			if userID == "" {
				// Fallback: try to get user by phone (for old JWT tokens without sub field)
				phone, _ := r.Context().Value(admin.CtxPhoneKey).(string)
				slog.Info("userID is empty, trying phone fallback", "phone", phone)
				if phone != "" {
					user, err := s.GetUserByPhone(phone)
					if err == nil && user != nil {
						userID = user.ID
						slog.Info("resolved userID from phone", "user_id", userID, "phone", phone)
					} else {
						slog.Error("failed to get user by phone", "phone", phone, "error", err)
					}
				}
			}
			if userID == "" {
				httputil.WriteError(w, http.StatusUnauthorized, "not authenticated", "auth_error", "no_user")
				return
			}

			member, err := s.GetTenantMember(tenantID, userID)
			if err != nil {
				httputil.WriteError(w, http.StatusForbidden, "not a member of this tenant", "auth_error", "not_member")
				return
			}

			if len(roles) > 0 {
				allowed := false
				for _, role := range roles {
					if member.Role == role {
						allowed = true
						break
					}
				}
				if !allowed {
					httputil.WriteError(w, http.StatusForbidden, "insufficient tenant permissions", "auth_error", "insufficient_tenant_role")
					return
				}
			}

			ctx := context.WithValue(r.Context(), admin.CtxTenantMemberKey, member)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractTenantID extracts the tenant ID from paths like /api/tenants/{id}/...
func extractTenantID(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, p := range parts {
		if p == "tenants" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
