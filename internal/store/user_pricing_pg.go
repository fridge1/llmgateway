package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

const userPricingSelect = `id, user_id, model_name, input_price, output_price,
	COALESCE(cached_input_price,0), COALESCE(cache_creation_price,0),
	COALESCE(cache_creation_1h_price,0), billing_type, is_active,
	COALESCE(pricing_tiers, 'null'::jsonb), COALESCE(created_by::text,''),
	discount_rate, created_at, updated_at`

func scanUserPricing(rows interface {
	Scan(dest ...any) error
}) (UserPricing, error) {
	var p UserPricing
	var tiersJSON []byte
	var discount sql.NullFloat64
	err := rows.Scan(
		&p.ID, &p.UserID, &p.ModelName,
		&p.InputPrice, &p.OutputPrice, &p.CachedInputPrice,
		&p.CacheCreationPrice, &p.CacheCreation1hPrice,
		&p.BillingType, &p.IsActive,
		&tiersJSON, &p.CreatedBy,
		&discount, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	if discount.Valid {
		v := discount.Float64
		p.DiscountRate = &v
	}
	if len(tiersJSON) > 0 && string(tiersJSON) != "null" {
		_ = json.Unmarshal(tiersJSON, &p.PricingTiers)
	}
	return p, nil
}

// ListUserPricing returns all pricing overrides for a user.
func (s *PgStore) ListUserPricing(userID string) ([]UserPricing, error) {
	rows, err := s.db.Query(
		`SELECT `+userPricingSelect+` FROM user_pricing WHERE user_id = $1 ORDER BY model_name`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list user pricing: %w", err)
	}
	defer rows.Close()

	var prices []UserPricing
	for rows.Next() {
		p, err := scanUserPricing(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan user pricing: %w", err)
		}
		prices = append(prices, p)
	}
	if prices == nil {
		prices = []UserPricing{}
	}
	return prices, nil
}

// GetUserPricing returns a specific user pricing override, or nil if not found.
func (s *PgStore) GetUserPricing(userID, modelName string) (*UserPricing, error) {
	row := s.db.QueryRow(
		`SELECT `+userPricingSelect+` FROM user_pricing WHERE user_id = $1 AND model_name = $2`,
		userID, modelName,
	)
	p, err := scanUserPricing(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user pricing: %w", err)
	}
	return &p, nil
}

// UpsertUserPricing inserts or updates a user pricing override.
func (s *PgStore) UpsertUserPricing(userID, modelName string, inputPrice, outputPrice, cachedInputPrice, cacheCreationPrice, cacheCreation1hPrice float64, billingType string, isActive bool, pricingTiers []PricingTier, discountRate *float64, createdBy string) error {
	if billingType == "" {
		billingType = "token"
	}

	var tiersJSON []byte
	if len(pricingTiers) > 0 {
		var err error
		tiersJSON, err = json.Marshal(pricingTiers)
		if err != nil {
			return fmt.Errorf("store: marshal user pricing tiers: %w", err)
		}
	}

	_, err := s.db.Exec(
		`INSERT INTO user_pricing (user_id, model_name,
		   input_price, output_price, cached_input_price,
		   cache_creation_price, cache_creation_1h_price,
		   billing_type, is_active, pricing_tiers, discount_rate, created_by, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		 ON CONFLICT (user_id, model_name) DO UPDATE SET
		   input_price = EXCLUDED.input_price,
		   output_price = EXCLUDED.output_price,
		   cached_input_price = EXCLUDED.cached_input_price,
		   cache_creation_price = EXCLUDED.cache_creation_price,
		   cache_creation_1h_price = EXCLUDED.cache_creation_1h_price,
		   billing_type = EXCLUDED.billing_type,
		   is_active = EXCLUDED.is_active,
		   pricing_tiers = EXCLUDED.pricing_tiers,
		   discount_rate = EXCLUDED.discount_rate,
		   created_by = EXCLUDED.created_by,
		   updated_at = NOW()`,
		userID, modelName, inputPrice, outputPrice, cachedInputPrice,
		cacheCreationPrice, cacheCreation1hPrice, billingType, isActive, tiersJSON, discountRate, createdBy,
	)
	if err != nil {
		return fmt.Errorf("store: upsert user pricing: %w", err)
	}
	return nil
}

// DeleteUserPricing removes a user pricing override.
func (s *PgStore) DeleteUserPricing(userID, modelName string) error {
	_, err := s.db.Exec(
		`DELETE FROM user_pricing WHERE user_id = $1 AND model_name = $2`,
		userID, modelName,
	)
	if err != nil {
		return fmt.Errorf("store: delete user pricing: %w", err)
	}
	return nil
}

// HasUserCustomPricing reports whether a user has any active pricing override.
func (s *PgStore) HasUserCustomPricing(userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM user_pricing WHERE user_id = $1 AND is_active = true)`,
		userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("store: has user custom pricing: %w", err)
	}
	return exists, nil
}
