package store

import "time"

// User represents a platform user.
type User struct {
	ID                        string     `json:"id"`
	Phone                     string     `json:"phone,omitempty"`
	Email                     string     `json:"email,omitempty"`
	PasswordHash              string     `json:"-"`
	Nickname                  string     `json:"nickname,omitempty"`
	Role                      string     `json:"role"`
	Status                    string     `json:"status"`
	EmailVerified             bool       `json:"email_verified,omitempty"`
	EmailVerifiedAt           *time.Time `json:"email_verified_at,omitempty"`
	FirstRechargeBonusGranted bool       `json:"-"`
	ImageShareEnabled         bool       `json:"image_share_enabled"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
}

// UserWithBalance extends User with the account balance.
type UserWithBalance struct {
	User
	Balance float64 `json:"balance"`
}

// AdminDashboardStats holds aggregate stats for the admin dashboard.
type AdminDashboardStats struct {
	TotalUsers    int     `json:"total_users"`
	TodayRevenue  float64 `json:"today_revenue"`
	TodayRequests int     `json:"today_requests"`
}

// ModelTokenStats holds per-model token consumption and cost breakdown.
type ModelTokenStats struct {
	Model               string  `json:"model"`
	PromptTokens        int64   `json:"prompt_tokens"`
	CompletionTokens    int64   `json:"completion_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	TotalCost           float64 `json:"total_cost"`
	PromptCost          float64 `json:"prompt_cost"`
	CompletionCost      float64 `json:"completion_cost"`
	CacheReadCost       float64 `json:"cache_read_cost"`
	CacheCreationCost   float64 `json:"cache_creation_cost"`
	RequestCount        int64   `json:"request_count"`
	BreakdownEstimated  bool    `json:"breakdown_estimated"`
	FailureCount        int64   `json:"failure_count"`
	SuccessRate         float64 `json:"success_rate"`
}

// AdminConsumptionStats holds global consumption statistics.
type AdminConsumptionStats struct {
	TotalCost     float64           `json:"total_cost"`
	TotalRequests int64             `json:"total_requests"`
	Models        []ModelTokenStats `json:"models"`
	DailyTrend    []DailyCost       `json:"daily_trend"`
}

// FunnelStage holds one stage of the acquisition→activation→retention funnel.
type FunnelStage struct {
	Key   string `json:"key"`   // stage identifier, e.g. "registered"
	Label string `json:"label"` // human-readable label
	Count int64  `json:"count"` // users reaching this stage
}

// AdminFunnelStats holds the conversion funnel over a time window:
// registered → first recharge (paid) → repeat recharge (≥2 paid) → active after first recharge.
// The cohort is users registered within the window [now-days, now].
type AdminFunnelStats struct {
	Days                int64         `json:"days"`
	Stages              []FunnelStage `json:"stages"`
	FirstRechargeRate   float64       `json:"first_recharge_rate"`
	RepeatRechargeRate  float64       `json:"repeat_recharge_rate"`
	PostRechargeUseRate float64       `json:"post_recharge_use_rate"`
}

// ImageDurationStats holds per-model image generation duration statistics.
type ImageDurationStats struct {
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	MinSeconds   float64 `json:"min_seconds"`
	AvgSeconds   float64 `json:"avg_seconds"`
	MaxSeconds   float64 `json:"max_seconds"`
}

// APIKey represents an API key belonging to a user.
type APIKey struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	KeyHash    string     `json:"-"`
	KeyPrefix  string     `json:"key_prefix"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	PlanID     *int       `json:"plan_id,omitempty"` // 非空时该 key 仅可访问该套餐内模型
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// UserPricing represents a user-specific model pricing override.
type UserPricing struct {
	ID                   int           `json:"id"`
	UserID               string        `json:"user_id"`
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

// ToModelPricing converts UserPricing to ModelPricing for cost calculation.
func (up *UserPricing) ToModelPricing() *ModelPricing {
	return &ModelPricing{
		ID:                   up.ID,
		ModelName:            up.ModelName,
		InputPrice:           up.InputPrice,
		OutputPrice:          up.OutputPrice,
		CachedInputPrice:     up.CachedInputPrice,
		CacheCreationPrice:   up.CacheCreationPrice,
		CacheCreation1hPrice: up.CacheCreation1hPrice,
		BillingType:          up.BillingType,
		IsActive:             up.IsActive,
		PricingTiers:         up.PricingTiers,
		UpdatedAt:            up.UpdatedAt,
	}
}
