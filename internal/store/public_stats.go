package store

import "fmt"

// PublicUsageTotals aggregates total request count and token usage across all users.
type PublicUsageTotals struct {
	TotalRequests int64
	TotalTokens   int64
}

// CountActiveUsers returns the number of users with status = 'active'.
func (s *PgStore) CountActiveUsers() (int64, error) {
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE status = 'active'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: count active users: %w", err)
	}
	return count, nil
}

// CountEnterpriseTenants returns the number of tenants flagged as enterprise.
func (s *PgStore) CountEnterpriseTenants() (int64, error) {
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM tenants WHERE is_enterprise = true`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: count enterprise tenants: %w", err)
	}
	return count, nil
}

// GetPublicUsageTotals returns the cumulative request count and token usage
// across consumption and subscription_usage transactions. Backed by
// idx_transactions_type_created (migration 000049).
func (s *PgStore) GetPublicUsageTotals() (*PublicUsageTotals, error) {
	var t PublicUsageTotals
	err := s.db.QueryRow(
		`SELECT COUNT(*),
		        COALESCE(SUM(COALESCE(prompt_tokens,0) + COALESCE(completion_tokens,0)
		                   + COALESCE(cache_read_tokens,0) + COALESCE(cache_creation_tokens,0)), 0)
		   FROM transactions
		  WHERE type IN ('consumption','subscription_usage')`,
	).Scan(&t.TotalRequests, &t.TotalTokens)
	if err != nil {
		return nil, fmt.Errorf("store: public usage totals: %w", err)
	}
	return &t, nil
}
