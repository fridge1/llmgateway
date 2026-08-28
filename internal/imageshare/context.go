package imageshare

import (
	"context"
	"net/http"

	"github.com/zhulang/llm-gateway/internal/admin"
)

// KeyIDFromContext returns the image-share key id from request context, set by the
// shared JWT middleware when the request carries an image_token.
func KeyIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(admin.CtxImageShareKeyID).(string)
	return v
}

// OwnerIDFromContext returns the owner user id of the image-share session.
func OwnerIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(admin.CtxImageShareOwnerID).(string)
	return v
}

// IsImageShareRequest reports whether the request is authenticated as an image-share session.
func IsImageShareRequest(r *http.Request) bool {
	role, _ := r.Context().Value(admin.CtxRoleKey).(string)
	return role == RoleImageShare
}
