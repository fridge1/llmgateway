package imageshare

import "time"

// Key represents an image-share key row.
type Key struct {
	ID          string     `json:"id"`
	OwnerUserID string     `json:"owner_user_id"`
	KeyHash     string     `json:"-"`
	KeyPrefix   string     `json:"key_prefix"`
	Name        string     `json:"name"`
	QuotaTotal  int        `json:"quota_total"`
	QuotaUsed   int        `json:"quota_used"`
	Status      string     `json:"status"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Remaining returns the remaining quota.
func (k *Key) Remaining() int {
	r := k.QuotaTotal - k.QuotaUsed
	if r < 0 {
		return 0
	}
	return r
}

// RoleImageShare is the JWT role value used for image-share sessions.
const RoleImageShare = "image_share"

// AllowedModel is the only model an image_share session may use.
const AllowedModel = "gpt-image-2"

// CookieName is the cookie name used to carry the image-share JWT.
const CookieName = "image_token"
