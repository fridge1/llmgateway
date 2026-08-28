package httputil

import (
	"context"
	"net/http"
)

// Identity 是一个可变的身份容器，在请求进入时由 AccessLog 中间件注入到 context，
// 认证成功后由代理认证逻辑回填。因为它是指针，handler 内部对其字段的写入对外层
// 中间件（在 ServeHTTP 返回后打日志）可见，从而把认证结果带出处理链。
type Identity struct {
	// Kind 标识身份类型：user / tenant / sub_user，未认证时为空。
	Kind string
	// Name 是用于日志展示的用户名：用户 key 为手机号，租户 key 为密钥名，子用户为用户名。
	Name string
	// UserID 是对应的主体 ID（用户 ID / 租户 ID / 子用户 ID）。
	UserID string
}

type identityCtxKey struct{}

// WithIdentitySlot 在 context 中放入一个空的身份容器并返回其指针，供后续认证逻辑回填。
func WithIdentitySlot(ctx context.Context) (context.Context, *Identity) {
	slot := &Identity{}
	return context.WithValue(ctx, identityCtxKey{}, slot), slot
}

// IdentityFromContext 取出 context 中的身份容器，未注入时返回 nil。
func IdentityFromContext(ctx context.Context) *Identity {
	slot, _ := ctx.Value(identityCtxKey{}).(*Identity)
	return slot
}

// SetIdentity 是回填身份的便捷方法：若 context 中存在身份容器则写入，否则无操作。
func SetIdentity(r *http.Request, kind, name, userID string) {
	if slot := IdentityFromContext(r.Context()); slot != nil {
		slot.Kind = kind
		slot.Name = name
		slot.UserID = userID
	}
}
