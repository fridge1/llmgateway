package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// PricingTier represents a single tier in tiered pricing.
// Prices are CNY per 1M tokens.
type PricingTier struct {
	MinTokens        int     `json:"min_tokens"`
	MaxTokens        int     `json:"max_tokens"`
	InputPrice       float64 `json:"input_price"`
	OutputPrice      float64 `json:"output_price"`
	CachedInputPrice float64 `json:"cached_input_price"`
}

// TimeBasedPricingRule defines a time window with a price multiplier.
// Days: 0=Sunday, 1=Monday, ..., 6=Saturday (matches time.Weekday).
// StartTime / EndTime: "HH:MM" in Asia/Shanghai (UTC+8).
// Multiplier: e.g. 1.5 = peak surcharge, 0.8 = off-peak discount.
// If EndTime < StartTime the window spans midnight (e.g. 22:00–06:00).
type TimeBasedPricingRule struct {
	Name       string  `json:"name"`
	Days       []int   `json:"days"`
	StartTime  string  `json:"start_time"`
	EndTime    string  `json:"end_time"`
	Multiplier float64 `json:"multiplier"`
}

// ModelPricing represents pricing for a model.
// For billing_type "token": input_price / output_price etc. are CNY per 1M tokens.
// For billing_type "image": input_price = per-image price for 1K/2K, output_price = per-image price for 4K.
// When PricingTiers is non-empty, calculateCost selects the tier matching the input token count.
type ModelPricing struct {
	ID                   int                    `json:"id"`
	ModelName            string                 `json:"model_name"`
	InputPrice           float64                `json:"input_price"`
	OutputPrice          float64                `json:"output_price"`
	CachedInputPrice     float64                `json:"cached_input_price"`
	CacheCreationPrice   float64                `json:"cache_creation_price"`
	CacheCreation1hPrice float64                `json:"cache_creation_1h_price"`
	BillingType          string                 `json:"billing_type"`
	IsActive             bool                   `json:"is_active"`
	UpdatedAt            time.Time              `json:"updated_at"`
	PricingTiers         []PricingTier          `json:"pricing_tiers,omitempty"`
	TimeBasedRules       []TimeBasedPricingRule `json:"time_based_rules,omitempty"`
}

const pricingSelect = `id, model_name, input_price, output_price, COALESCE(cached_input_price,0),
	COALESCE(cache_creation_price,0), COALESCE(cache_creation_1h_price,0),
	billing_type, is_active, updated_at,
	COALESCE(pricing_tiers, 'null'::jsonb),
	COALESCE(time_based_rules, 'null'::jsonb)`

func scanModelPricing(rows interface {
	Scan(dest ...any) error
}) (ModelPricing, error) {
	var p ModelPricing
	var tiersJSON, rulesJSON []byte
	err := rows.Scan(
		&p.ID, &p.ModelName, &p.InputPrice, &p.OutputPrice, &p.CachedInputPrice,
		&p.CacheCreationPrice, &p.CacheCreation1hPrice,
		&p.BillingType, &p.IsActive, &p.UpdatedAt,
		&tiersJSON, &rulesJSON,
	)
	if err != nil {
		return p, err
	}
	if len(tiersJSON) > 0 && string(tiersJSON) != "null" {
		_ = json.Unmarshal(tiersJSON, &p.PricingTiers)
	}
	if len(rulesJSON) > 0 && string(rulesJSON) != "null" {
		_ = json.Unmarshal(rulesJSON, &p.TimeBasedRules)
	}
	return p, nil
}

// ListPricing returns all model pricing records.
func (s *PgStore) ListPricing() ([]ModelPricing, error) {
	rows, err := s.db.Query(
		`SELECT ` + pricingSelect + ` FROM model_pricing ORDER BY model_name`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list pricing: %w", err)
	}
	defer rows.Close()

	var prices []ModelPricing
	for rows.Next() {
		p, err := scanModelPricing(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan pricing: %w", err)
		}
		prices = append(prices, p)
	}
	if prices == nil {
		prices = []ModelPricing{}
	}
	return prices, nil
}

// ListActivePricing returns only active model pricing records.
func (s *PgStore) ListActivePricing() ([]ModelPricing, error) {
	rows, err := s.db.Query(
		`SELECT `+pricingSelect+` FROM model_pricing WHERE is_active = true ORDER BY model_name`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list active pricing: %w", err)
	}
	defer rows.Close()

	var prices []ModelPricing
	for rows.Next() {
		p, err := scanModelPricing(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan pricing: %w", err)
		}
		prices = append(prices, p)
	}
	if prices == nil {
		prices = []ModelPricing{}
	}
	return prices, nil
}

// UpsertPricing inserts or updates pricing. All prices are in CNY.
func (s *PgStore) UpsertPricing(modelName string, inputPrice, outputPrice, cachedInputPrice, cacheCreationPrice, cacheCreation1hPrice float64, billingType string, isActive bool, pricingTiers []PricingTier, timeBasedRules []TimeBasedPricingRule) error {
	if billingType == "" {
		billingType = "token"
	}

	var tiersJSON []byte
	if len(pricingTiers) > 0 {
		var err error
		tiersJSON, err = json.Marshal(pricingTiers)
		if err != nil {
			return fmt.Errorf("store: marshal pricing tiers: %w", err)
		}
	}

	var rulesJSON []byte
	if len(timeBasedRules) > 0 {
		var err error
		rulesJSON, err = json.Marshal(timeBasedRules)
		if err != nil {
			return fmt.Errorf("store: marshal time based rules: %w", err)
		}
	}

	_, err := s.db.Exec(
		`INSERT INTO model_pricing (model_name,
		   input_price, output_price, cached_input_price, cache_creation_price, cache_creation_1h_price,
		   billing_type, is_active, pricing_tiers, time_based_rules, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		 ON CONFLICT (model_name) DO UPDATE SET
		   input_price = EXCLUDED.input_price,
		   output_price = EXCLUDED.output_price,
		   cached_input_price = EXCLUDED.cached_input_price,
		   cache_creation_price = EXCLUDED.cache_creation_price,
		   cache_creation_1h_price = EXCLUDED.cache_creation_1h_price,
		   billing_type = EXCLUDED.billing_type,
		   is_active = EXCLUDED.is_active,
		   pricing_tiers = EXCLUDED.pricing_tiers,
		   time_based_rules = EXCLUDED.time_based_rules,
		   updated_at = NOW()`,
		modelName, inputPrice, outputPrice, cachedInputPrice, cacheCreationPrice, cacheCreation1hPrice,
		billingType, isActive, tiersJSON, rulesJSON,
	)
	if err != nil {
		return fmt.Errorf("store: upsert pricing: %w", err)
	}
	return nil
}

// GetPricing returns pricing for a specific model.
func (s *PgStore) GetPricing(modelName string) (*ModelPricing, error) {
	row := s.db.QueryRow(
		`SELECT `+pricingSelect+` FROM model_pricing WHERE model_name = $1`,
		modelName,
	)
	p, err := scanModelPricing(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: pricing not found for model %s", modelName)
	}
	if err != nil {
		return nil, fmt.Errorf("store: get pricing: %w", err)
	}
	return &p, nil
}

// PricingChangeLog records a pricing or FX rate change.
type PricingChangeLog struct {
	ID          int64          `json:"id"`
	ModelName   string         `json:"model_name"`
	ChangeType  string         `json:"change_type"`
	AdminUserID string         `json:"admin_user_id"`
	OldValues   map[string]any `json:"old_values"`
	NewValues   map[string]any `json:"new_values"`
	CreatedAt   time.Time      `json:"created_at"`
}

// InsertPricingChangeLog records a pricing change event.
func (s *PgStore) InsertPricingChangeLog(modelName, changeType, adminUserID string, oldValues, newValues map[string]any) error {
	oldJSON, _ := json.Marshal(oldValues)
	newJSON, _ := json.Marshal(newValues)
	_, err := s.db.Exec(
		`INSERT INTO pricing_change_logs (model_name, change_type, admin_user_id, old_values, new_values)
		 VALUES ($1, $2, $3, $4, $5)`,
		modelName, changeType, adminUserID, oldJSON, newJSON,
	)
	if err != nil {
		return fmt.Errorf("store: insert pricing change log: %w", err)
	}
	return nil
}

// ListPricingChangeLogs returns paginated pricing change logs.
func (s *PgStore) ListPricingChangeLogs(limit, offset int) ([]PricingChangeLog, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pricing_change_logs`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count pricing change logs: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT id, model_name, change_type, COALESCE(admin_user_id,''), old_values, new_values, created_at
		 FROM pricing_change_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list pricing change logs: %w", err)
	}
	defer rows.Close()

	var logs []PricingChangeLog
	for rows.Next() {
		var l PricingChangeLog
		var oldJSON, newJSON []byte
		if err := rows.Scan(&l.ID, &l.ModelName, &l.ChangeType, &l.AdminUserID, &oldJSON, &newJSON, &l.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan pricing change log: %w", err)
		}
		if len(oldJSON) > 0 {
			_ = json.Unmarshal(oldJSON, &l.OldValues)
		}
		if len(newJSON) > 0 {
			_ = json.Unmarshal(newJSON, &l.NewValues)
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []PricingChangeLog{}
	}
	return logs, total, nil
}
