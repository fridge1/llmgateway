package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

const tenantPricingSelect = `id, tenant_id, model_name, input_price, output_price,
	COALESCE(cached_input_price,0), COALESCE(cache_creation_price,0),
	COALESCE(cache_creation_1h_price,0), billing_type, is_active,
	COALESCE(pricing_tiers, 'null'::jsonb), COALESCE(created_by::text,''),
	discount_rate, created_at, updated_at`

func scanTenantPricing(rows interface {
	Scan(dest ...any) error
}) (TenantPricing, error) {
	var p TenantPricing
	var tiersJSON []byte
	var discount sql.NullFloat64
	err := rows.Scan(
		&p.ID, &p.TenantID, &p.ModelName,
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

// ListTenantPricing returns all pricing overrides for a tenant.
func (s *PgStore) ListTenantPricing(tenantID string) ([]TenantPricing, error) {
	rows, err := s.db.Query(
		`SELECT `+tenantPricingSelect+` FROM tenant_pricing WHERE tenant_id = $1 ORDER BY model_name`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list tenant pricing: %w", err)
	}
	defer rows.Close()

	var prices []TenantPricing
	for rows.Next() {
		p, err := scanTenantPricing(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan tenant pricing: %w", err)
		}
		prices = append(prices, p)
	}
	if prices == nil {
		prices = []TenantPricing{}
	}
	return prices, nil
}

// GetTenantPricing returns a specific tenant pricing override, or error if not found.
func (s *PgStore) GetTenantPricing(tenantID, modelName string) (*TenantPricing, error) {
	row := s.db.QueryRow(
		`SELECT `+tenantPricingSelect+` FROM tenant_pricing WHERE tenant_id = $1 AND model_name = $2`,
		tenantID, modelName,
	)
	p, err := scanTenantPricing(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get tenant pricing: %w", err)
	}
	return &p, nil
}

// UpsertTenantPricing inserts or updates a tenant pricing override.
func (s *PgStore) UpsertTenantPricing(tenantID, modelName string, inputPrice, outputPrice, cachedInputPrice, cacheCreationPrice, cacheCreation1hPrice float64, billingType string, isActive bool, pricingTiers []PricingTier, discountRate *float64, createdBy string) error {
	if billingType == "" {
		billingType = "token"
	}

	var tiersJSON []byte
	if len(pricingTiers) > 0 {
		var err error
		tiersJSON, err = json.Marshal(pricingTiers)
		if err != nil {
			return fmt.Errorf("store: marshal tenant pricing tiers: %w", err)
		}
	}

	_, err := s.db.Exec(
		`INSERT INTO tenant_pricing (tenant_id, model_name,
		   input_price, output_price, cached_input_price,
		   cache_creation_price, cache_creation_1h_price,
		   billing_type, is_active, pricing_tiers, discount_rate, created_by, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		 ON CONFLICT (tenant_id, model_name) DO UPDATE SET
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
		tenantID, modelName, inputPrice, outputPrice, cachedInputPrice,
		cacheCreationPrice, cacheCreation1hPrice, billingType, isActive, tiersJSON, discountRate, createdBy,
	)
	if err != nil {
		return fmt.Errorf("store: upsert tenant pricing: %w", err)
	}
	return nil
}

// DeleteTenantPricing removes a tenant pricing override.
func (s *PgStore) DeleteTenantPricing(tenantID, modelName string) error {
	_, err := s.db.Exec(
		`DELETE FROM tenant_pricing WHERE tenant_id = $1 AND model_name = $2`,
		tenantID, modelName,
	)
	if err != nil {
		return fmt.Errorf("store: delete tenant pricing: %w", err)
	}
	return nil
}

// HasTenantCustomPricing reports whether a tenant has any active pricing override.
func (s *PgStore) HasTenantCustomPricing(tenantID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM tenant_pricing WHERE tenant_id = $1 AND is_active = true)`,
		tenantID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("store: has tenant custom pricing: %w", err)
	}
	return exists, nil
}

// GetUserPrimaryPricingTenant returns the tenant_id of the user's earliest-joined
// active tenant that has at least one active pricing override. Returns "" if none.
func (s *PgStore) GetUserPrimaryPricingTenant(userID string) (string, error) {
	var tenantID string
	err := s.db.QueryRow(
		`SELECT t.id
		 FROM tenants t
		 JOIN tenant_members tm ON t.id = tm.tenant_id
		 WHERE tm.user_id = $1 AND t.status = 'active'
		   AND EXISTS(SELECT 1 FROM tenant_pricing tp WHERE tp.tenant_id = t.id AND tp.is_active = true)
		 ORDER BY tm.joined_at
		 LIMIT 1`,
		userID,
	).Scan(&tenantID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("store: get user primary pricing tenant: %w", err)
	}
	return tenantID, nil
}
