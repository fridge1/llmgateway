package proxy

import (
	"github.com/zhulang/llm-gateway/internal/moderation"
)

// Moderation is the optional content-safety service; nil disables screening.
// Set once at startup before requests flow (no lock needed).
func (c *Core) SetModeration(m *moderation.Service) { c.moderation = m }

// ModerationIdentity resolves the (userID, tenantID) pair used for moderation
// applicability checks and hit records.
func ModerationIdentity(auth *AuthResult) (userID, tenantID string) {
	switch {
	case auth.IsSubUser():
		return auth.SubUser.ID, auth.SubUser.TenantID
	case auth.IsTenant():
		return "", auth.TenantKey.TenantID
	case auth.User != nil:
		return auth.User.ID, auth.MemberTenantID
	}
	return "", ""
}

// CheckModeration screens text for the given auth/model. It returns the
// verdict; callers reject with their protocol-specific error envelope.
// A nil service or empty text short-circuits to a clean verdict.
func (c *Core) CheckModeration(auth *AuthResult, model, text string) moderation.Verdict {
	if c.moderation == nil || text == "" {
		return moderation.Verdict{}
	}
	userID, tenantID := ModerationIdentity(auth)
	if !c.moderation.Applicable(model, tenantID) {
		return moderation.Verdict{}
	}
	return c.moderation.CheckAndRecord(text, model, userID, tenantID)
}
