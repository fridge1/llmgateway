package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/money"
)

// SubscriptionPlan represents a subscription tier definition.
type SubscriptionPlan struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	DisplayName     string    `json:"display_name"`
	Description     string    `json:"description"`
	MonthlyPriceCNY float64   `json:"monthly_price_cny"`
	QuotaAmountCNY  float64   `json:"quota_amount_cny"`
	DurationDays    int       `json:"duration_days"`
	SortOrder       int       `json:"sort_order"`
	Status          string    `json:"status"`
	Models          []string  `json:"models,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UserSubscription represents an active or past user subscription.
type UserSubscription struct {
	ID            string            `json:"id"`
	UserID        string            `json:"user_id"`
	PlanID        int               `json:"plan_id"`
	Plan          *SubscriptionPlan `json:"plan,omitempty"`
	Status        string            `json:"status"`
	Brand         string            `json:"brand"`
	StartedAt     time.Time         `json:"started_at"`
	ExpiresAt     time.Time         `json:"expires_at"`
	AutoRenew     bool              `json:"auto_renew"`
	ExtraQuotaCNY float64           `json:"extra_quota_cny"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// SubscriptionUsageDetail holds per-model usage within a subscription period.
type SubscriptionUsageDetail struct {
	ModelName               string  `json:"model_name"`
	InputTokensUsed         int64   `json:"input_tokens_used"`
	OutputTokensUsed        int64   `json:"output_tokens_used"`
	CacheReadTokensUsed     int64   `json:"cache_read_tokens_used"`
	CacheCreationTokensUsed int64   `json:"cache_creation_tokens_used"`
	AmountUsed              float64 `json:"amount_used"`
	RequestCount            int     `json:"request_count"`
}

// SubscriptionUsageSummary aggregates usage across all models for a subscription period.
type SubscriptionUsageSummary struct {
	TotalAmountUsed float64                   `json:"total_amount_used"`
	QuotaAmountCNY  float64                   `json:"quota_amount_cny"`
	UsagePercent    float64                   `json:"usage_percent"`
	ModelDetails    []SubscriptionUsageDetail `json:"model_details"`
}

// SubscriptionOrder represents a payment record for a subscription.
type SubscriptionOrder struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	PlanID        int        `json:"plan_id"`
	AmountCNY     float64    `json:"amount_cny"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	PaymentMethod *string    `json:"payment_method,omitempty"`
	PaymentID     *string    `json:"payment_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
}

// SubscriptionOrderDaily holds daily aggregation for subscription orders.
type SubscriptionOrderDaily struct {
	Date   string  `json:"date"`
	Count  int     `json:"count"`
	Amount float64 `json:"amount"`
}

// SubscriptionOrderByPlan holds per-plan aggregation for subscription orders.
type SubscriptionOrderByPlan struct {
	PlanName string  `json:"plan_name"`
	Count    int     `json:"count"`
	Amount   float64 `json:"amount"`
}

// SubscriptionOrderByType holds per-type aggregation for subscription orders.
type SubscriptionOrderByType struct {
	Type   string  `json:"type"`
	Count  int     `json:"count"`
	Amount float64 `json:"amount"`
}

// SubscriptionOrderStats holds aggregated subscription order statistics.
type SubscriptionOrderStats struct {
	TotalRevenue  float64                  `json:"total_revenue"`
	TotalOrders   int                      `json:"total_orders"`
	PaidOrders    int                      `json:"paid_orders"`
	PendingOrders int                      `json:"pending_orders"`
	AvgOrderValue float64                  `json:"avg_order_value"`
	DailyTrend    []SubscriptionOrderDaily `json:"daily_trend"`
	PlanBreakdown []SubscriptionOrderByPlan `json:"plan_breakdown"`
	TypeBreakdown []SubscriptionOrderByType `json:"type_breakdown"`
}

// AdminSubscriptionOrder is a subscription order with user identifier and plan name for admin listing.
type AdminSubscriptionOrder struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	UserIdentifier string     `json:"user_identifier"`
	PlanID         int        `json:"plan_id"`
	PlanName       string     `json:"plan_name"`
	AmountCNY      float64    `json:"amount_cny"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	PaymentMethod  *string    `json:"payment_method,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
}

// AdminSubscriptionUserUsage represents a subscription user with usage details for admin listing.
type AdminSubscriptionUserUsage struct {
	SubscriptionID  string    `json:"subscription_id"`
	UserID          string    `json:"user_id"`
	UserIdentifier  string    `json:"user_identifier"`
	UserNickname    string    `json:"user_nickname"`
	PlanID          int       `json:"plan_id"`
	PlanName        string    `json:"plan_name"`
	PlanCategory    string    `json:"plan_category"` // "image" | "openai" | "claude"
	PlanPriceCNY    float64   `json:"plan_price_cny"`
	QuotaAmountCNY  float64   `json:"quota_amount_cny"`
	AmountUsed      float64   `json:"amount_used"`
	AmountRemaining float64   `json:"amount_remaining"`
	UsagePercent    float64   `json:"usage_percent"`
	RequestCount    int       `json:"request_count"`
	Status          string    `json:"status"`
	StartedAt       time.Time `json:"started_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	AutoRenew       bool      `json:"auto_renew"`
}

// ListSubscriptionPlans returns all active subscription plans ordered by sort_order.
func (s *PgStore) ListSubscriptionPlans() ([]SubscriptionPlan, error) {
	rows, err := s.db.Query(
		`SELECT id, name, display_name, COALESCE(description,''), monthly_price_cny, quota_amount_cny,
		        duration_days, sort_order, status, created_at, updated_at
		 FROM subscription_plans WHERE status = 'active' ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("store: list subscription plans: %w", err)
	}
	defer rows.Close()

	var plans []SubscriptionPlan
	for rows.Next() {
		var p SubscriptionPlan
		if err := rows.Scan(&p.ID, &p.Name, &p.DisplayName, &p.Description,
			&p.MonthlyPriceCNY, &p.QuotaAmountCNY, &p.DurationDays, &p.SortOrder, &p.Status,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan subscription plan: %w", err)
		}
		plans = append(plans, p)
	}

	// Load models for each plan.
	for i := range plans {
		mrows, err := s.db.Query(
			`SELECT model_pattern FROM subscription_plan_models WHERE plan_id = $1`, plans[i].ID)
		if err != nil {
			return nil, fmt.Errorf("store: list plan models: %w", err)
		}
		for mrows.Next() {
			var m string
			if err := mrows.Scan(&m); err != nil {
				mrows.Close()
				return nil, fmt.Errorf("store: scan plan model: %w", err)
			}
			plans[i].Models = append(plans[i].Models, m)
		}
		mrows.Close()
	}

	if plans == nil {
		plans = []SubscriptionPlan{}
	}
	return plans, nil
}

// GetSubscriptionPlan returns a plan by ID.
func (s *PgStore) GetSubscriptionPlan(id int) (*SubscriptionPlan, error) {
	var p SubscriptionPlan
	err := s.db.QueryRow(
		`SELECT id, name, display_name, COALESCE(description,''), monthly_price_cny, quota_amount_cny,
		        duration_days, sort_order, status, created_at, updated_at
		 FROM subscription_plans WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.DisplayName, &p.Description,
		&p.MonthlyPriceCNY, &p.QuotaAmountCNY, &p.DurationDays, &p.SortOrder, &p.Status,
		&p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: subscription plan not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get subscription plan: %w", err)
	}
	return &p, nil
}

// GetSubscriptionPlanModels returns the model patterns for a plan.
func (s *PgStore) GetSubscriptionPlanModels(planID int) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT model_pattern FROM subscription_plan_models WHERE plan_id = $1`, planID)
	if err != nil {
		return nil, fmt.Errorf("store: get plan models: %w", err)
	}
	defer rows.Close()
	var models []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, fmt.Errorf("store: scan plan model: %w", err)
		}
		models = append(models, m)
	}
	return models, nil
}

// ListAllSubscriptionPlans returns all subscription plans (regardless of status)
// ordered by sort_order, with their models loaded. For admin management.
func (s *PgStore) ListAllSubscriptionPlans() ([]SubscriptionPlan, error) {
	rows, err := s.db.Query(
		`SELECT id, name, display_name, COALESCE(description,''), monthly_price_cny, quota_amount_cny,
		        duration_days, sort_order, status, created_at, updated_at
		 FROM subscription_plans ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("store: list all subscription plans: %w", err)
	}
	defer rows.Close()

	var plans []SubscriptionPlan
	for rows.Next() {
		var p SubscriptionPlan
		if err := rows.Scan(&p.ID, &p.Name, &p.DisplayName, &p.Description,
			&p.MonthlyPriceCNY, &p.QuotaAmountCNY, &p.DurationDays, &p.SortOrder, &p.Status,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan subscription plan: %w", err)
		}
		plans = append(plans, p)
	}

	// Load models for each plan.
	for i := range plans {
		mrows, err := s.db.Query(
			`SELECT model_pattern FROM subscription_plan_models WHERE plan_id = $1`, plans[i].ID)
		if err != nil {
			return nil, fmt.Errorf("store: list plan models: %w", err)
		}
		for mrows.Next() {
			var m string
			if err := mrows.Scan(&m); err != nil {
				mrows.Close()
				return nil, fmt.Errorf("store: scan plan model: %w", err)
			}
			plans[i].Models = append(plans[i].Models, m)
		}
		mrows.Close()
	}

	if plans == nil {
		plans = []SubscriptionPlan{}
	}
	return plans, nil
}

// CreateSubscriptionPlan inserts a new subscription plan and returns it.
// Model associations are managed separately via SetSubscriptionPlanModels.
func (s *PgStore) CreateSubscriptionPlan(p SubscriptionPlan) (*SubscriptionPlan, error) {
	if p.Status == "" {
		p.Status = "active"
	}
	err := s.db.QueryRow(
		`INSERT INTO subscription_plans
		   (name, display_name, description, monthly_price_cny, quota_amount_cny,
		    duration_days, sort_order, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at, updated_at`,
		p.Name, p.DisplayName, p.Description, p.MonthlyPriceCNY, p.QuotaAmountCNY,
		p.DurationDays, p.SortOrder, p.Status,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create subscription plan: %w", err)
	}
	return &p, nil
}

// UpdateSubscriptionPlan updates a plan's metadata by ID.
// Model associations are managed separately via SetSubscriptionPlanModels.
func (s *PgStore) UpdateSubscriptionPlan(p SubscriptionPlan) error {
	res, err := s.db.Exec(
		`UPDATE subscription_plans
		 SET name = $2, display_name = $3, description = $4, monthly_price_cny = $5,
		     quota_amount_cny = $6, duration_days = $7, sort_order = $8, status = $9,
		     updated_at = NOW()
		 WHERE id = $1`,
		p.ID, p.Name, p.DisplayName, p.Description, p.MonthlyPriceCNY,
		p.QuotaAmountCNY, p.DurationDays, p.SortOrder, p.Status,
	)
	if err != nil {
		return fmt.Errorf("store: update subscription plan: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("store: subscription plan not found")
	}
	return nil
}

// DeleteSubscriptionPlan removes a plan by ID. Its model associations are removed
// automatically via ON DELETE CASCADE on subscription_plan_models.
func (s *PgStore) DeleteSubscriptionPlan(id int) error {
	res, err := s.db.Exec(`DELETE FROM subscription_plans WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete subscription plan: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("store: subscription plan not found")
	}
	return nil
}

// SetSubscriptionPlanModels replaces the full set of model associations for a plan.
// It runs in a transaction: delete all existing rows, then insert the deduped models.
func (s *PgStore) SetSubscriptionPlanModels(planID int, models []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin set plan models: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM subscription_plan_models WHERE plan_id = $1`, planID); err != nil {
		return fmt.Errorf("store: clear plan models: %w", err)
	}

	seen := make(map[string]bool)
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		if _, err := tx.Exec(
			`INSERT INTO subscription_plan_models (plan_id, model_pattern)
			 VALUES ($1, $2) ON CONFLICT (plan_id, model_pattern) DO NOTHING`,
			planID, m); err != nil {
			return fmt.Errorf("store: insert plan model: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit set plan models: %w", err)
	}
	return nil
}

// GetActiveSubscription returns the user's current active subscription with plan details.
func (s *PgStore) GetActiveSubscription(userID string) (*UserSubscription, error) {
	var sub UserSubscription
	var p SubscriptionPlan
	err := s.db.QueryRow(
		`SELECT us.id, us.user_id, us.plan_id, us.status, us.brand, us.started_at, us.expires_at,
		        us.auto_renew, us.extra_quota_cny, us.created_at, us.updated_at,
		        sp.id, sp.name, sp.display_name, COALESCE(sp.description,''),
		        sp.monthly_price_cny, sp.quota_amount_cny, sp.duration_days, sp.sort_order, sp.status
		 FROM user_subscriptions us
		 JOIN subscription_plans sp ON sp.id = us.plan_id
		 WHERE us.user_id = $1 AND us.status = 'active' AND us.expires_at > NOW()`,
		userID,
	).Scan(&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.Brand, &sub.StartedAt, &sub.ExpiresAt,
		&sub.AutoRenew, &sub.ExtraQuotaCNY, &sub.CreatedAt, &sub.UpdatedAt,
		&p.ID, &p.Name, &p.DisplayName, &p.Description,
		&p.MonthlyPriceCNY, &p.QuotaAmountCNY, &p.DurationDays, &p.SortOrder, &p.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get active subscription: %w", err)
	}
	sub.Plan = &p
	return &sub, nil
}

// GetActiveSubscriptions returns all active subscriptions for the user.
func (s *PgStore) GetActiveSubscriptions(userID string) ([]UserSubscription, error) {
	rows, err := s.db.Query(
		`SELECT us.id, us.user_id, us.plan_id, us.status, us.brand, us.started_at, us.expires_at,
		        us.auto_renew, us.extra_quota_cny, us.created_at, us.updated_at,
		        sp.id, sp.name, sp.display_name, COALESCE(sp.description,''),
		        sp.monthly_price_cny, sp.quota_amount_cny, sp.duration_days, sp.sort_order, sp.status
		 FROM user_subscriptions us
		 JOIN subscription_plans sp ON sp.id = us.plan_id
		 WHERE us.user_id = $1 AND us.status = 'active' AND us.expires_at > NOW()
		 ORDER BY us.started_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get active subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []UserSubscription
	for rows.Next() {
		var sub UserSubscription
		var p SubscriptionPlan
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.Brand, &sub.StartedAt, &sub.ExpiresAt,
			&sub.AutoRenew, &sub.ExtraQuotaCNY, &sub.CreatedAt, &sub.UpdatedAt,
			&p.ID, &p.Name, &p.DisplayName, &p.Description,
			&p.MonthlyPriceCNY, &p.QuotaAmountCNY, &p.DurationDays, &p.SortOrder, &p.Status); err != nil {
			return nil, fmt.Errorf("store: scan active subscription: %w", err)
		}
		sub.Plan = &p
		subs = append(subs, sub)
	}
	return subs, nil
}

// GetActiveSubscriptionByBrand returns the user's active subscription for a specific brand.
func (s *PgStore) GetActiveSubscriptionByBrand(userID, brand string) (*UserSubscription, error) {
	namePattern := "openai-%"
	if brand != "openai" {
		namePattern = ""
	}

	var query string
	if brand == "openai" {
		query = `SELECT us.id, us.user_id, us.plan_id, us.status, us.brand, us.started_at, us.expires_at,
		        us.auto_renew, us.extra_quota_cny, us.created_at, us.updated_at,
		        sp.id, sp.name, sp.display_name, COALESCE(sp.description,''),
		        sp.monthly_price_cny, sp.quota_amount_cny, sp.duration_days, sp.sort_order, sp.status
		 FROM user_subscriptions us
		 JOIN subscription_plans sp ON sp.id = us.plan_id
		 WHERE us.user_id = $1 AND us.status = 'active' AND us.expires_at > NOW() AND sp.name LIKE $2`
	} else {
		query = `SELECT us.id, us.user_id, us.plan_id, us.status, us.brand, us.started_at, us.expires_at,
		        us.auto_renew, us.extra_quota_cny, us.created_at, us.updated_at,
		        sp.id, sp.name, sp.display_name, COALESCE(sp.description,''),
		        sp.monthly_price_cny, sp.quota_amount_cny, sp.duration_days, sp.sort_order, sp.status
		 FROM user_subscriptions us
		 JOIN subscription_plans sp ON sp.id = us.plan_id
		 WHERE us.user_id = $1 AND us.status = 'active' AND us.expires_at > NOW() AND sp.name NOT LIKE 'openai-%'`
	}

	var sub UserSubscription
	var p SubscriptionPlan
	var err error
	if brand == "openai" {
		err = s.db.QueryRow(query, userID, namePattern).Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.Brand, &sub.StartedAt, &sub.ExpiresAt,
			&sub.AutoRenew, &sub.ExtraQuotaCNY, &sub.CreatedAt, &sub.UpdatedAt,
			&p.ID, &p.Name, &p.DisplayName, &p.Description,
			&p.MonthlyPriceCNY, &p.QuotaAmountCNY, &p.DurationDays, &p.SortOrder, &p.Status)
	} else {
		err = s.db.QueryRow(query, userID).Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.Brand, &sub.StartedAt, &sub.ExpiresAt,
			&sub.AutoRenew, &sub.ExtraQuotaCNY, &sub.CreatedAt, &sub.UpdatedAt,
			&p.ID, &p.Name, &p.DisplayName, &p.Description,
			&p.MonthlyPriceCNY, &p.QuotaAmountCNY, &p.DurationDays, &p.SortOrder, &p.Status)
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get active subscription by brand: %w", err)
	}
	sub.Plan = &p
	return &sub, nil
}

// CreateSubscription creates a new subscription for the user.
func (s *PgStore) CreateSubscription(userID string, planID int, expiresAt time.Time, brand string) (*UserSubscription, error) {
	var sub UserSubscription
	err := s.db.QueryRow(
		`INSERT INTO user_subscriptions (user_id, plan_id, expires_at, brand)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, plan_id, status, brand, started_at, expires_at, auto_renew, created_at, updated_at`,
		userID, planID, expiresAt, brand,
	).Scan(&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.Brand, &sub.StartedAt, &sub.ExpiresAt,
		&sub.AutoRenew, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create subscription: %w", err)
	}
	return &sub, nil
}

// UpgradeSubscriptionParams carries inputs for an atomic upgrade.
type UpgradeSubscriptionParams struct {
	UserID            string
	OldSubscriptionID string
	NewPlanID         int
	Brand             string
	PriceCNY          float64
	ExtraQuotaCNY     float64
	DeductDesc        string
	StartedAt         time.Time
	ExpiresAt         time.Time
}

// UpgradeSubscriptionTx performs an atomic plan upgrade: deducts balance,
// expires the old subscription row, inserts a fresh one with carried-over
// extra quota, and writes a paid upgrade order — all in a single DB tx.
// A fresh subscription_id ensures subscription_usage ledgers do not bleed
// across plans when the user upgrades on the same calendar day.
func (s *PgStore) UpgradeSubscriptionTx(p UpgradeSubscriptionParams) (*UserSubscription, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin upgrade tx: %w", err)
	}
	defer tx.Rollback()

	var bal, frozen float64
	if err := tx.QueryRow(
		`SELECT balance, frozen FROM balances WHERE user_id = $1 FOR UPDATE`,
		p.UserID,
	).Scan(&bal, &frozen); err != nil {
		return nil, fmt.Errorf("store: upgrade read balance: %w", err)
	}
	// Tolerant comparison, consistent with DeductForSubscription.
	if !money.GTE(bal-frozen, p.PriceCNY) {
		return nil, fmt.Errorf("余额不足：可用余额 ¥%.2f，需要 ¥%.2f", bal-frozen, p.PriceCNY)
	}

	if _, err := tx.Exec(
		`UPDATE balances SET balance = GREATEST(balance - $1, 0), updated_at = NOW() WHERE user_id = $2`,
		p.PriceCNY, p.UserID,
	); err != nil {
		return nil, fmt.Errorf("store: upgrade deduct balance: %w", err)
	}

	var balanceAfter float64
	if err := tx.QueryRow(
		`SELECT balance FROM balances WHERE user_id = $1`, p.UserID,
	).Scan(&balanceAfter); err != nil {
		return nil, fmt.Errorf("store: upgrade read balance after: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO transactions (user_id, type, amount, balance_after, description)
		 VALUES ($1, 'sub_purchase', $2, $3, $4)`,
		p.UserID, p.PriceCNY, balanceAfter, p.DeductDesc,
	); err != nil {
		return nil, fmt.Errorf("store: upgrade insert transaction: %w", err)
	}

	res, err := tx.Exec(
		`UPDATE user_subscriptions SET status = 'expired', updated_at = NOW()
		 WHERE id = $1 AND status = 'active'`,
		p.OldSubscriptionID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: upgrade expire old sub: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("store: old subscription not found or not active")
	}

	var sub UserSubscription
	if err := tx.QueryRow(
		`INSERT INTO user_subscriptions (user_id, plan_id, started_at, expires_at, brand, extra_quota_cny)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, plan_id, status, brand, started_at, expires_at, auto_renew, extra_quota_cny, created_at, updated_at`,
		p.UserID, p.NewPlanID, p.StartedAt, p.ExpiresAt, p.Brand, p.ExtraQuotaCNY,
	).Scan(&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.Brand, &sub.StartedAt, &sub.ExpiresAt,
		&sub.AutoRenew, &sub.ExtraQuotaCNY, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
		return nil, fmt.Errorf("store: upgrade create new sub: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO subscription_orders (user_id, plan_id, amount_cny, type, status, payment_method, paid_at)
		 VALUES ($1, $2, $3, 'upgrade', 'paid', 'balance', NOW())`,
		p.UserID, p.NewPlanID, p.PriceCNY,
	); err != nil {
		return nil, fmt.Errorf("store: upgrade insert order: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: upgrade commit: %w", err)
	}
	return &sub, nil
}

// AddSubscriptionQuota atomically adds quota to an active subscription.
func (s *PgStore) AddSubscriptionQuota(subscriptionID string, additionalQuota float64) error {
	res, err := s.db.Exec(
		`UPDATE user_subscriptions SET extra_quota_cny = extra_quota_cny + $1, updated_at = NOW()
		 WHERE id = $2 AND status = 'active'`,
		additionalQuota, subscriptionID,
	)
	if err != nil {
		return fmt.Errorf("store: add subscription quota: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("store: subscription not found or not active")
	}
	return nil
}

// CancelSubscription sets auto_renew to false.
func (s *PgStore) CancelSubscription(subscriptionID string) error {
	res, err := s.db.Exec(
		`UPDATE user_subscriptions SET auto_renew = false, updated_at = NOW()
		 WHERE id = $1 AND status = 'active'`,
		subscriptionID,
	)
	if err != nil {
		return fmt.Errorf("store: cancel subscription: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("store: subscription not found or not active")
	}
	return nil
}

// ResumeSubscription sets auto_renew to true.
func (s *PgStore) ResumeSubscription(subscriptionID string) error {
	res, err := s.db.Exec(
		`UPDATE user_subscriptions SET auto_renew = true, updated_at = NOW()
		 WHERE id = $1 AND status = 'active'`,
		subscriptionID,
	)
	if err != nil {
		return fmt.Errorf("store: resume subscription: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("store: subscription not found or not active")
	}
	return nil
}

// ExpireSubscription marks a subscription as expired.
func (s *PgStore) ExpireSubscription(subscriptionID string) error {
	_, err := s.db.Exec(
		`UPDATE user_subscriptions SET status = 'expired', updated_at = NOW() WHERE id = $1`,
		subscriptionID,
	)
	if err != nil {
		return fmt.Errorf("store: expire subscription: %w", err)
	}
	return nil
}

// ExpireExpiredSubscriptions batch-expires all active subscriptions past their expires_at.
func (s *PgStore) ExpireExpiredSubscriptions() (int, error) {
	result, err := s.db.Exec(
		`UPDATE user_subscriptions SET status = 'expired', updated_at = NOW()
		 WHERE status = 'active' AND expires_at <= NOW()`,
	)
	if err != nil {
		return 0, fmt.Errorf("store: expire expired subscriptions: %w", err)
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// ExpireUserSubscriptionsByBrand expires active subscriptions for a user+brand that are past expires_at.
func (s *PgStore) ExpireUserSubscriptionsByBrand(userID, brand string) error {
	var query string
	if brand == "openai" {
		query = `UPDATE user_subscriptions SET status = 'expired', updated_at = NOW()
			 WHERE user_id = $1 AND status = 'active' AND expires_at <= NOW()
			   AND plan_id IN (SELECT id FROM subscription_plans WHERE name LIKE 'openai-%')`
	} else {
		query = `UPDATE user_subscriptions SET status = 'expired', updated_at = NOW()
			 WHERE user_id = $1 AND status = 'active' AND expires_at <= NOW()
			   AND plan_id IN (SELECT id FROM subscription_plans WHERE name NOT LIKE 'openai-%')`
	}
	_, err := s.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("store: expire user subscriptions by brand: %w", err)
	}
	return nil
}

// GetSubscriptionTotalUsage returns the total amount used for a subscription in a given period.
func (s *PgStore) GetSubscriptionTotalUsage(subscriptionID string, period time.Time) (float64, error) {
	var total float64
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(amount_used), 0) FROM subscription_usage
		 WHERE subscription_id = $1 AND period = $2`,
		subscriptionID, period,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("store: get subscription total usage: %w", err)
	}
	return total, nil
}

// IncrementSubscriptionUsage atomically increments usage counters for a model in a subscription period.
func (s *PgStore) IncrementSubscriptionUsage(subscriptionID, userID, model string, period time.Time, tokens TokenUsage, amount float64) error {
	_, err := s.db.Exec(
		`INSERT INTO subscription_usage (subscription_id, user_id, model_name, period,
		    input_tokens_used, output_tokens_used, cache_read_tokens_used, cache_creation_tokens_used,
		    amount_used, request_count, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1, NOW())
		 ON CONFLICT (subscription_id, model_name, period)
		 DO UPDATE SET
		    input_tokens_used = subscription_usage.input_tokens_used + EXCLUDED.input_tokens_used,
		    output_tokens_used = subscription_usage.output_tokens_used + EXCLUDED.output_tokens_used,
		    cache_read_tokens_used = subscription_usage.cache_read_tokens_used + EXCLUDED.cache_read_tokens_used,
		    cache_creation_tokens_used = subscription_usage.cache_creation_tokens_used + EXCLUDED.cache_creation_tokens_used,
		    amount_used = subscription_usage.amount_used + EXCLUDED.amount_used,
		    request_count = subscription_usage.request_count + 1,
		    updated_at = NOW()`,
		subscriptionID, userID, model, period,
		tokens.PromptTokens, tokens.CompletionTokens, tokens.CacheReadTokens,
		tokens.CacheCreationTokens+tokens.CacheCreation5mTokens+tokens.CacheCreation1hTokens,
		amount,
	)
	if err != nil {
		return fmt.Errorf("store: increment subscription usage: %w", err)
	}
	return nil
}

// RecordSubscriptionUsageAndTransaction atomically increments subscription usage
// and records the corresponding transaction in a single database transaction.
// amount: 实际交易金额（元），记录到 transactions.amount
// quotaConsumed: 配额消耗（图片套餐=张数，普通套餐=金额），记录到 subscription_usage.amount_used
func (s *PgStore) RecordSubscriptionUsageAndTransaction(subscriptionID, userID, model, requestID string, period time.Time, tokens TokenUsage, amount float64, quotaConsumed float64, apiKeyID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin subscription usage tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Increment subscription_usage (使用 quotaConsumed 作为配额消耗)
	_, err = tx.Exec(
		`INSERT INTO subscription_usage (subscription_id, user_id, model_name, period,
		    input_tokens_used, output_tokens_used, cache_read_tokens_used, cache_creation_tokens_used,
		    amount_used, request_count, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1, NOW())
		 ON CONFLICT (subscription_id, model_name, period)
		 DO UPDATE SET
		    input_tokens_used = subscription_usage.input_tokens_used + EXCLUDED.input_tokens_used,
		    output_tokens_used = subscription_usage.output_tokens_used + EXCLUDED.output_tokens_used,
		    cache_read_tokens_used = subscription_usage.cache_read_tokens_used + EXCLUDED.cache_read_tokens_used,
		    cache_creation_tokens_used = subscription_usage.cache_creation_tokens_used + EXCLUDED.cache_creation_tokens_used,
		    amount_used = subscription_usage.amount_used + EXCLUDED.amount_used,
		    request_count = subscription_usage.request_count + 1,
		    updated_at = NOW()`,
		subscriptionID, userID, model, period,
		tokens.PromptTokens, tokens.CompletionTokens, tokens.CacheReadTokens,
		tokens.CacheCreationTokens+tokens.CacheCreation5mTokens+tokens.CacheCreation1hTokens,
		quotaConsumed,
	)
	if err != nil {
		return fmt.Errorf("store: increment subscription usage (tx): %w", err)
	}

	// 2. Record subscription_usage transaction (使用 amount 作为实际交易金额)
	var balanceAfter float64
	if err := tx.QueryRow(
		`SELECT balance FROM balances WHERE user_id = $1`, userID,
	).Scan(&balanceAfter); err != nil {
		return fmt.Errorf("store: subscription tx get balance (tx): %w", err)
	}

	var apiKeyIDPtr *string
	if apiKeyID != "" {
		apiKeyIDPtr = &apiKeyID
	}

	_, err = tx.Exec(
		`INSERT INTO transactions (user_id, type, amount, balance_after, model, request_id,
		        prompt_tokens, completion_tokens, cache_read_tokens,
		        cache_creation_tokens, cache_creation_5m_tokens, cache_creation_1h_tokens,
		        subscription_id, api_key_id)
		 VALUES ($1, 'subscription_usage', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		userID, amount, balanceAfter, model, requestID,
		intPtrOrNil(tokens.PromptTokens), intPtrOrNil(tokens.CompletionTokens),
		intPtrOrNil(tokens.CacheReadTokens), intPtrOrNil(tokens.CacheCreationTokens),
		intPtrOrNil(tokens.CacheCreation5mTokens), intPtrOrNil(tokens.CacheCreation1hTokens),
		subscriptionID, apiKeyIDPtr,
	)
	if err != nil {
		return fmt.Errorf("store: insert subscription transaction (tx): %w", err)
	}

	return tx.Commit()
}

// GetSubscriptionUsageSummary returns aggregated usage for a subscription period.
func (s *PgStore) GetSubscriptionUsageSummary(subscriptionID string, period time.Time, quotaAmount float64) (*SubscriptionUsageSummary, error) {
	rows, err := s.db.Query(
		`SELECT model_name, input_tokens_used, output_tokens_used,
		        cache_read_tokens_used, cache_creation_tokens_used,
		        amount_used, request_count
		 FROM subscription_usage
		 WHERE subscription_id = $1 AND period = $2
		 ORDER BY amount_used DESC`,
		subscriptionID, period,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get subscription usage summary: %w", err)
	}
	defer rows.Close()

	summary := &SubscriptionUsageSummary{QuotaAmountCNY: quotaAmount}
	for rows.Next() {
		var d SubscriptionUsageDetail
		if err := rows.Scan(&d.ModelName, &d.InputTokensUsed, &d.OutputTokensUsed,
			&d.CacheReadTokensUsed, &d.CacheCreationTokensUsed,
			&d.AmountUsed, &d.RequestCount); err != nil {
			return nil, fmt.Errorf("store: scan subscription usage: %w", err)
		}
		summary.TotalAmountUsed += d.AmountUsed
		summary.ModelDetails = append(summary.ModelDetails, d)
	}
	if summary.ModelDetails == nil {
		summary.ModelDetails = []SubscriptionUsageDetail{}
	}
	if quotaAmount > 0 {
		summary.UsagePercent = summary.TotalAmountUsed / quotaAmount * 100
		if summary.UsagePercent > 100 {
			summary.UsagePercent = 100
		}
	}
	return summary, nil
}

// CreateSubscriptionOrder creates a payment order for a subscription.
func (s *PgStore) CreateSubscriptionOrder(userID string, planID int, amountCNY float64, orderType string) (*SubscriptionOrder, error) {
	var o SubscriptionOrder
	err := s.db.QueryRow(
		`INSERT INTO subscription_orders (user_id, plan_id, amount_cny, type)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, plan_id, amount_cny, type, status, payment_method, payment_id, created_at, paid_at`,
		userID, planID, amountCNY, orderType,
	).Scan(&o.ID, &o.UserID, &o.PlanID, &o.AmountCNY, &o.Type, &o.Status,
		&o.PaymentMethod, &o.PaymentID, &o.CreatedAt, &o.PaidAt)
	if err != nil {
		return nil, fmt.Errorf("store: create subscription order: %w", err)
	}
	return &o, nil
}

// CompleteSubscriptionOrder marks an order as paid.
func (s *PgStore) CompleteSubscriptionOrder(orderID, paymentMethod, paymentID string) error {
	res, err := s.db.Exec(
		`UPDATE subscription_orders SET status = 'paid', payment_method = $1, payment_id = $2, paid_at = NOW()
		 WHERE id = $3 AND status = 'pending'`,
		paymentMethod, paymentID, orderID,
	)
	if err != nil {
		return fmt.Errorf("store: complete subscription order: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("store: subscription order not found or already completed")
	}
	return nil
}

// ListUserSubscriptionHistory returns all subscriptions (active and historical)
// for the given user, ordered by created_at DESC. Caps the result at 200 rows
// to avoid pathological responses.
func (s *PgStore) ListUserSubscriptionHistory(userID string) ([]UserSubscription, error) {
	rows, err := s.db.Query(
		`SELECT us.id, us.user_id, us.plan_id, us.status, us.brand, us.started_at, us.expires_at,
		        us.auto_renew, us.extra_quota_cny, us.created_at, us.updated_at,
		        sp.id, sp.name, sp.display_name, COALESCE(sp.description,''),
		        sp.monthly_price_cny, sp.quota_amount_cny, sp.duration_days, sp.sort_order, sp.status
		 FROM user_subscriptions us
		 JOIN subscription_plans sp ON sp.id = us.plan_id
		 WHERE us.user_id = $1
		 ORDER BY us.created_at DESC
		 LIMIT 200`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list user subscription history: %w", err)
	}
	defer rows.Close()

	var subs []UserSubscription
	for rows.Next() {
		var sub UserSubscription
		var p SubscriptionPlan
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.Brand, &sub.StartedAt, &sub.ExpiresAt,
			&sub.AutoRenew, &sub.ExtraQuotaCNY, &sub.CreatedAt, &sub.UpdatedAt,
			&p.ID, &p.Name, &p.DisplayName, &p.Description,
			&p.MonthlyPriceCNY, &p.QuotaAmountCNY, &p.DurationDays, &p.SortOrder, &p.Status); err != nil {
			return nil, fmt.Errorf("store: scan user subscription history: %w", err)
		}
		sub.Plan = &p
		subs = append(subs, sub)
	}
	if subs == nil {
		subs = []UserSubscription{}
	}
	return subs, nil
}

// ListUserSubscriptions returns all subscriptions for admin listing.
func (s *PgStore) ListUserSubscriptions(limit, offset int) ([]UserSubscription, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM user_subscriptions`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count subscriptions: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT us.id, us.user_id, us.plan_id, us.status, us.brand, us.started_at, us.expires_at,
		        us.auto_renew, us.created_at, us.updated_at,
		        sp.id, sp.name, sp.display_name, COALESCE(sp.description,''),
		        sp.monthly_price_cny, sp.quota_amount_cny, sp.duration_days, sp.sort_order, sp.status
		 FROM user_subscriptions us
		 JOIN subscription_plans sp ON sp.id = us.plan_id
		 ORDER BY us.created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []UserSubscription
	for rows.Next() {
		var sub UserSubscription
		var p SubscriptionPlan
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.Brand, &sub.StartedAt, &sub.ExpiresAt,
			&sub.AutoRenew, &sub.CreatedAt, &sub.UpdatedAt,
			&p.ID, &p.Name, &p.DisplayName, &p.Description,
			&p.MonthlyPriceCNY, &p.QuotaAmountCNY, &p.DurationDays, &p.SortOrder, &p.Status); err != nil {
			return nil, 0, fmt.Errorf("store: scan subscription: %w", err)
		}
		sub.Plan = &p
		subs = append(subs, sub)
	}
	if subs == nil {
		subs = []UserSubscription{}
	}
	return subs, total, nil
}

// GetSubscriptionOrderStats returns aggregated subscription order statistics for the given number of days.
func (s *PgStore) GetSubscriptionOrderStats(days int) (*SubscriptionOrderStats, error) {
	stats := &SubscriptionOrderStats{}

	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(CASE WHEN status='paid' THEN amount_cny ELSE 0 END), 0),
		        COUNT(*),
		        COUNT(*) FILTER (WHERE status='paid'),
		        COUNT(*) FILTER (WHERE status='pending'),
		        COALESCE(AVG(CASE WHEN status='paid' THEN amount_cny END), 0)
		 FROM subscription_orders WHERE created_at >= NOW() - $1 * INTERVAL '1 day'`, days,
	).Scan(&stats.TotalRevenue, &stats.TotalOrders, &stats.PaidOrders, &stats.PendingOrders, &stats.AvgOrderValue)
	if err != nil {
		return nil, fmt.Errorf("store: subscription order stats totals: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT d::date::text AS date,
		        COUNT(so.id)::int AS count,
		        COALESCE(SUM(so.amount_cny), 0) AS amount
		 FROM generate_series(
		     (NOW() - $1 * INTERVAL '1 day')::date,
		     NOW()::date,
		     '1 day'
		 ) d
		 LEFT JOIN subscription_orders so ON so.created_at::date = d::date AND so.status = 'paid'
		 GROUP BY d ORDER BY d`, days)
	if err != nil {
		return nil, fmt.Errorf("store: subscription order stats daily: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d SubscriptionOrderDaily
		if err := rows.Scan(&d.Date, &d.Count, &d.Amount); err != nil {
			return nil, fmt.Errorf("store: scan daily trend: %w", err)
		}
		stats.DailyTrend = append(stats.DailyTrend, d)
	}
	if stats.DailyTrend == nil {
		stats.DailyTrend = []SubscriptionOrderDaily{}
	}

	planRows, err := s.db.Query(
		`SELECT sp.display_name, COUNT(so.id)::int, COALESCE(SUM(so.amount_cny), 0)
		 FROM subscription_orders so
		 JOIN subscription_plans sp ON sp.id = so.plan_id
		 WHERE so.created_at >= NOW() - $1 * INTERVAL '1 day' AND so.status = 'paid'
		 GROUP BY sp.display_name ORDER BY SUM(so.amount_cny) DESC`, days)
	if err != nil {
		return nil, fmt.Errorf("store: subscription order stats by plan: %w", err)
	}
	defer planRows.Close()
	for planRows.Next() {
		var p SubscriptionOrderByPlan
		if err := planRows.Scan(&p.PlanName, &p.Count, &p.Amount); err != nil {
			return nil, fmt.Errorf("store: scan plan breakdown: %w", err)
		}
		stats.PlanBreakdown = append(stats.PlanBreakdown, p)
	}
	if stats.PlanBreakdown == nil {
		stats.PlanBreakdown = []SubscriptionOrderByPlan{}
	}

	typeRows, err := s.db.Query(
		`SELECT type, COUNT(*)::int, COALESCE(SUM(amount_cny), 0)
		 FROM subscription_orders
		 WHERE created_at >= NOW() - $1 * INTERVAL '1 day' AND status = 'paid'
		 GROUP BY type ORDER BY type`, days)
	if err != nil {
		return nil, fmt.Errorf("store: subscription order stats by type: %w", err)
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var t SubscriptionOrderByType
		if err := typeRows.Scan(&t.Type, &t.Count, &t.Amount); err != nil {
			return nil, fmt.Errorf("store: scan type breakdown: %w", err)
		}
		stats.TypeBreakdown = append(stats.TypeBreakdown, t)
	}
	if stats.TypeBreakdown == nil {
		stats.TypeBreakdown = []SubscriptionOrderByType{}
	}

	return stats, nil
}

// ListAllSubscriptionOrders returns a paginated list of subscription orders with user phone and plan name.
func (s *PgStore) ListAllSubscriptionOrders(limit, offset int, status, orderType string) ([]AdminSubscriptionOrder, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if status != "" {
		where += fmt.Sprintf(" AND so.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if orderType != "" {
		where += fmt.Sprintf(" AND so.type = $%d", argIdx)
		args = append(args, orderType)
		argIdx++
	}

	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM subscription_orders so %s`, where)
	if err := s.db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count subscription orders: %w", err)
	}

	query := fmt.Sprintf(
		`SELECT so.id, so.user_id, COALESCE(u.phone, u.email, NULLIF(u.nickname, ''), SUBSTRING(u.id::text, 1, 8)), so.plan_id, COALESCE(sp.display_name,''),
		        so.amount_cny, so.type, so.status, so.payment_method, so.created_at, so.paid_at
		 FROM subscription_orders so
		 LEFT JOIN users u ON u.id = so.user_id
		 LEFT JOIN subscription_plans sp ON sp.id = so.plan_id
		 %s ORDER BY so.created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list subscription orders: %w", err)
	}
	defer rows.Close()

	var orders []AdminSubscriptionOrder
	for rows.Next() {
		var o AdminSubscriptionOrder
		if err := rows.Scan(&o.ID, &o.UserID, &o.UserIdentifier, &o.PlanID, &o.PlanName,
			&o.AmountCNY, &o.Type, &o.Status, &o.PaymentMethod, &o.CreatedAt, &o.PaidAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan subscription order: %w", err)
		}
		orders = append(orders, o)
	}
	if orders == nil {
		orders = []AdminSubscriptionOrder{}
	}
	return orders, total, nil
}

// ListSubscriptionUsersUsage returns a paginated list of subscription users with their usage details.
func (s *PgStore) ListSubscriptionUsersUsage(limit, offset int, search, status, planFilter string) ([]AdminSubscriptionUserUsage, int, int, float64, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if search != "" {
		where += fmt.Sprintf(" AND (u.phone ILIKE $%d OR u.email ILIKE $%d OR u.nickname ILIKE $%d)", argIdx, argIdx, argIdx)
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern)
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND us.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if planFilter != "" {
		where += fmt.Sprintf(" AND us.plan_id = $%d", argIdx)
		args = append(args, planFilter)
		argIdx++
	}

	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM user_subscriptions us
		JOIN users u ON u.id = us.user_id
		JOIN subscription_plans sp ON sp.id = us.plan_id
		%s`, where)
	if err := s.db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("store: count subscription users: %w", err)
	}

	var activeCount int
	var totalUsage float64
	// 本月总用量按"套餐内单价"统计，而非上游模型原价：
	// 图片套餐 amount_used 存的是张数，套餐内单价 = monthly_price_cny / quota_amount_cny
	// （如 ¥300/2000张 = ¥0.15/张），用量金额 = 张数 × 套餐内单价；
	// 普通套餐 amount_used 本身即金额，直接累加。
	summaryQ := fmt.Sprintf(`SELECT
		COUNT(*) FILTER (WHERE us.status = 'active'),
		COALESCE(SUM(
			CASE
				WHEN sp.name LIKE 'image-%%' AND sp.quota_amount_cny > 0
					THEN su_agg.total_used * sp.monthly_price_cny / sp.quota_amount_cny
				ELSE su_agg.total_used
			END
		), 0)
		FROM user_subscriptions us
		JOIN users u ON u.id = us.user_id
		JOIN subscription_plans sp ON sp.id = us.plan_id
		LEFT JOIN (
			SELECT subscription_id, period, SUM(amount_used) AS total_used
			FROM subscription_usage
			GROUP BY subscription_id, period
		) su_agg ON su_agg.subscription_id = us.id AND su_agg.period = (us.started_at AT TIME ZONE 'UTC')::date
		%s`, where)
	if err := s.db.QueryRow(summaryQ, args...).Scan(&activeCount, &totalUsage); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("store: get subscription summary: %w", err)
	}

	query := fmt.Sprintf(`SELECT
		us.id,
		us.user_id,
		COALESCE(u.phone, u.email, NULLIF(u.nickname, ''), SUBSTRING(u.id::text, 1, 8)),
		COALESCE(u.nickname, ''),
		us.plan_id,
		COALESCE(sp.display_name, sp.name),
		sp.name,
		sp.monthly_price_cny,
		sp.quota_amount_cny + us.extra_quota_cny,
		COALESCE(su_agg.total_used, 0),
		GREATEST(sp.quota_amount_cny + us.extra_quota_cny - COALESCE(su_agg.total_used, 0), 0),
		CASE WHEN sp.quota_amount_cny + us.extra_quota_cny > 0
			THEN LEAST(COALESCE(su_agg.total_used, 0) / (sp.quota_amount_cny + us.extra_quota_cny) * 100, 100)
			ELSE 0 END,
		COALESCE(su_agg.total_reqs, 0)::int,
		us.status,
		us.started_at,
		us.expires_at,
		us.auto_renew
		FROM user_subscriptions us
		JOIN users u ON u.id = us.user_id
		JOIN subscription_plans sp ON sp.id = us.plan_id
		LEFT JOIN (
			SELECT subscription_id, period,
				SUM(amount_used) AS total_used,
				SUM(request_count) AS total_reqs
			FROM subscription_usage
			GROUP BY subscription_id, period
		) su_agg ON su_agg.subscription_id = us.id AND su_agg.period = (us.started_at AT TIME ZONE 'UTC')::date
		%s ORDER BY su_agg.total_used DESC NULLS LAST LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1)
	dataArgs := append(args, limit, offset)

	rows, err := s.db.Query(query, dataArgs...)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("store: list subscription users usage: %w", err)
	}
	defer rows.Close()

	var users []AdminSubscriptionUserUsage
	for rows.Next() {
		var u AdminSubscriptionUserUsage
		var rawPlanName string
		if err := rows.Scan(&u.SubscriptionID, &u.UserID, &u.UserIdentifier, &u.UserNickname,
			&u.PlanID, &u.PlanName, &rawPlanName, &u.PlanPriceCNY, &u.QuotaAmountCNY,
			&u.AmountUsed, &u.AmountRemaining, &u.UsagePercent, &u.RequestCount,
			&u.Status, &u.StartedAt, &u.ExpiresAt, &u.AutoRenew); err != nil {
			return nil, 0, 0, 0, fmt.Errorf("store: scan subscription user usage: %w", err)
		}
		u.PlanCategory = planCategoryFromName(rawPlanName)
		users = append(users, u)
	}
	if users == nil {
		users = []AdminSubscriptionUserUsage{}
	}
	return users, total, activeCount, totalUsage, nil
}

// planCategoryFromName 根据套餐 name 前缀返回类别：image / openai / claude。
func planCategoryFromName(name string) string {
	switch {
	case strings.HasPrefix(name, "image-"):
		return "image"
	case strings.HasPrefix(name, "openai-"):
		return "openai"
	default:
		return "claude"
	}
}
