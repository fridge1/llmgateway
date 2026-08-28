package store

import "time"

// Tenant represents an organization/company that groups users and shares resources.
type Tenant struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	OwnerID        string    `json:"owner_id"`
	Status         string    `json:"status"`
	IsEnterprise   bool      `json:"is_enterprise"`
	CreatedByAdmin *string   `json:"created_by_admin,omitempty"`
	ContactPhone   string    `json:"contact_phone,omitempty"`
	ContactEmail   string    `json:"contact_email,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TenantMember represents a user's membership in a tenant.
type TenantMember struct {
	ID       string    `json:"id"`
	TenantID string    `json:"tenant_id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
	Phone    string    `json:"phone,omitempty"`
	Nickname string    `json:"nickname,omitempty"`
}

// TenantBalance holds the financial state of a tenant.
type TenantBalance struct {
	TenantID       string  `json:"tenant_id"`
	Balance        float64 `json:"balance"`
	Frozen         float64 `json:"frozen"`
	TotalRecharged float64 `json:"total_recharged"`
	TotalConsumed  float64 `json:"total_consumed"`
}

// TenantAPIKey represents an API key owned by a tenant.
type TenantAPIKey struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	KeyPrefix  string     `json:"key_prefix"`
	CreatedBy  string     `json:"created_by"`
	Status     string     `json:"status"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// TenantInvitation represents a pending invitation to join a tenant.
type TenantInvitation struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	TenantName string    `json:"tenant_name,omitempty"`
	Phone      string    `json:"phone"`
	Role       string    `json:"role"`
	InvitedBy  string    `json:"invited_by"`
	Status     string    `json:"status"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// TenantWithRole is returned when listing tenants for a user.
type TenantWithRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// TenantSubUser represents a sub-user within a tenant (not counted in platform user statistics).
type TenantSubUser struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Username   string     `json:"username"`
	Nickname   string     `json:"nickname,omitempty"`
	Status     string     `json:"status"`
	QuotaLimit *float64   `json:"quota_limit"`  // NULL = unlimited
	QuotaUsed  float64    `json:"quota_used"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// TenantSubUserKey represents an API key owned by a tenant sub-user.
type TenantSubUserKey struct {
	ID         string     `json:"id"`
	SubUserID  string     `json:"sub_user_id"`
	TenantID   string     `json:"tenant_id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	KeyPrefix  string     `json:"key_prefix"`
	Status     string     `json:"status"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// TenantSubUserWithQuota includes quota information for display.
type TenantSubUserWithQuota struct {
	TenantSubUser
	QuotaRemaining *float64 `json:"quota_remaining"` // NULL = unlimited
}

// TenantModelUpstream is a tenant-specific upstream override for one model.
// When a tenant has rows for a model, requests using that tenant's keys route
// exclusively to these upstreams (no fallback to the global pool).
type TenantModelUpstream struct {
	ID               int64     `json:"id"`
	TenantID         string    `json:"tenant_id"`
	ModelName        string    `json:"model_name"`
	Provider         string    `json:"provider"`
	Protocol         string    `json:"protocol"`
	Protocols        []string  `json:"protocols"`
	UpstreamProvider string    `json:"upstream_provider"`
	UpstreamName     string    `json:"upstream_name"`
	BaseURL          string    `json:"base_url"`
	APIKey           string    `json:"api_key"`
	ModelOverride    string    `json:"model_override"`
	Weight           int       `json:"weight"`
	SortOrder        int       `json:"sort_order"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TenantPricing represents a tenant-specific model pricing override.
type TenantPricing struct {
	ID                   int           `json:"id"`
	TenantID             string        `json:"tenant_id"`
	ModelName            string        `json:"model_name"`
	InputPrice           float64       `json:"input_price"`
	OutputPrice          float64       `json:"output_price"`
	CachedInputPrice     float64       `json:"cached_input_price"`
	CacheCreationPrice   float64       `json:"cache_creation_price"`
	CacheCreation1hPrice float64       `json:"cache_creation_1h_price"`
	BillingType          string        `json:"billing_type"`
	IsActive             bool          `json:"is_active"`
	PricingTiers         []PricingTier `json:"pricing_tiers,omitempty"`
	// DiscountRate, when non-nil, makes the effective price = global price × rate.
	// When nil, the absolute prices above are used (legacy behavior).
	DiscountRate         *float64      `json:"discount_rate,omitempty"`
	CreatedBy            string        `json:"created_by,omitempty"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
}

// ToModelPricing converts a TenantPricing to a ModelPricing for cost calculation.
func (tp *TenantPricing) ToModelPricing() *ModelPricing {
	return &ModelPricing{
		ID:                   tp.ID,
		ModelName:            tp.ModelName,
		InputPrice:           tp.InputPrice,
		OutputPrice:          tp.OutputPrice,
		CachedInputPrice:     tp.CachedInputPrice,
		CacheCreationPrice:   tp.CacheCreationPrice,
		CacheCreation1hPrice: tp.CacheCreation1hPrice,
		BillingType:          tp.BillingType,
		IsActive:             tp.IsActive,
		UpdatedAt:            tp.UpdatedAt,
		PricingTiers:         tp.PricingTiers,
	}
}
