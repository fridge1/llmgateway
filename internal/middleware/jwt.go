package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
)

// RequireRole returns a middleware that checks the JWT role claim against allowed roles.
// Must be used after JWTAuth.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(admin.CtxRoleKey).(string)
			for _, allowed := range roles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			httputil.WriteError(w, http.StatusForbidden, "insufficient permissions", "auth_error", "insufficient_role")
		})
	}
}

// imageShareAllowedPrefixes lists the only API paths an image_share session may access.
// Anything outside this list is rejected even though the JWT is valid.
var imageShareAllowedPrefixes = []string{
	"/api/image-share/me",
	"/api/image-share/logout",
	"/api/image/tasks",
	"/api/image/models",
}

// JWTAuth returns a middleware that validates JWT from cookie or Authorization header.
func JWTAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""
			source := ""

			if strings.HasPrefix(r.URL.Path, "/api/sub-user/") {
				if c, err := r.Cookie("sub_token"); err == nil {
					tokenStr = c.Value
					source = "sub_token"
				}
				if tokenStr == "" {
					if c, err := r.Cookie("token"); err == nil {
						tokenStr = c.Value
						source = "token"
					}
				}
			} else {
				// Image-share endpoints should accept the dedicated image_token cookie
				// instead of the regular session token.
				if strings.HasPrefix(r.URL.Path, "/api/image-share/") || strings.HasPrefix(r.URL.Path, "/api/image/") {
					if c, err := r.Cookie("image_token"); err == nil {
						tokenStr = c.Value
						source = "image_token"
					}
				}
				if tokenStr == "" {
					if c, err := r.Cookie("token"); err == nil {
						tokenStr = c.Value
						source = "token"
					}
				}
				if tokenStr == "" {
					if c, err := r.Cookie("sub_token"); err == nil {
						tokenStr = c.Value
						source = "sub_token"
					}
				}
			}

			// Fall back to Authorization header.
			if tokenStr == "" {
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					tokenStr = strings.TrimPrefix(auth, "Bearer ")
					source = "bearer"
				}
			}

			if tokenStr == "" {
				httputil.WriteError(w, http.StatusUnauthorized, "not logged in", "auth_error", "no_token")
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				httputil.WriteError(w, http.StatusUnauthorized, "session expired", "auth_error", "invalid_token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				httputil.WriteError(w, http.StatusUnauthorized, "invalid token claims", "auth_error", "bad_claims")
				return
			}

			phone, _ := claims["phone"].(string)
			role, _ := claims["role"].(string)
			sub, _ := claims["sub"].(string)

			// image_share sessions:
			//   - sub claim is the image-share key id (NOT a user id)
			//   - owner_user_id claim carries the user that owns the key
			//   - the cookie used to deliver the token must be image_token
			//   - access is restricted to a small allowlist of paths
			if role == "image_share" {
				if source != "image_token" && source != "bearer" {
					httputil.WriteError(w, http.StatusForbidden, "wrong token source", "auth_error", "wrong_source")
					return
				}
				allowed := false
				for _, p := range imageShareAllowedPrefixes {
					if r.URL.Path == p || strings.HasPrefix(r.URL.Path, p+"/") {
						allowed = true
						break
					}
				}
				if !allowed {
					httputil.WriteError(w, http.StatusForbidden, "endpoint not allowed for image_share", "auth_error", "share_endpoint_forbidden")
					return
				}
				ownerID, _ := claims["owner_user_id"].(string)
				if sub == "" || ownerID == "" {
					httputil.WriteError(w, http.StatusUnauthorized, "invalid token claims", "auth_error", "bad_claims")
					return
				}
				ctx := r.Context()
				ctx = context.WithValue(ctx, admin.CtxRoleKey, role)
				// Surface owner_user_id as the canonical user_id so downstream handlers
				// charge owner balance and own sessions under the owner.
				ctx = context.WithValue(ctx, admin.CtxUserIDKey, ownerID)
				ctx = context.WithValue(ctx, admin.CtxImageShareKeyID, sub)
				ctx = context.WithValue(ctx, admin.CtxImageShareOwnerID, ownerID)
				httputil.SetIdentity(r, "image_share", ownerID, ownerID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Reject regular tokens that arrived via the image_token cookie.
			if source == "image_token" {
				httputil.WriteError(w, http.StatusForbidden, "wrong token type", "auth_error", "wrong_token_type")
				return
			}

			// Set both old and new context keys for backward compat
			ctx := r.Context()
			ctx = context.WithValue(ctx, admin.CtxUsernameKey, phone)
			ctx = context.WithValue(ctx, admin.CtxPhoneKey, phone)
			ctx = context.WithValue(ctx, admin.CtxRoleKey, role)
			ctx = context.WithValue(ctx, admin.CtxUserIDKey, sub)
			httputil.SetIdentity(r, "user", phone, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
