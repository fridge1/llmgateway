package store

import (
	"fmt"
)

// WeeklyUsageSummary is a per-user aggregate of the last 7 days of usage,
// used to build the Monday usage-report notification.
type WeeklyUsageSummary struct {
	UserID       string
	RequestCount int64
	TotalTokens  int64
	TotalCost    float64
}

// GetWeeklyUsageSummaries returns one row per user that had any billable
// consumption in the last 7 days. A single grouped query avoids N per-user
// stat calls when fanning out the weekly report.
func (s *PgStore) GetWeeklyUsageSummaries() ([]WeeklyUsageSummary, error) {
	rows, err := s.db.Query(
		`SELECT user_id,
		        COUNT(*) AS request_count,
		        COALESCE(SUM(COALESCE(prompt_tokens,0) + COALESCE(completion_tokens,0)
		                   + COALESCE(cache_read_tokens,0) + COALESCE(cache_creation_tokens,0)), 0) AS total_tokens,
		        COALESCE(SUM(amount), 0) AS total_cost
		 FROM transactions
		 WHERE type IN ('consumption', 'subscription_usage')
		   AND created_at >= NOW() - INTERVAL '7 days'
		 GROUP BY user_id
		 HAVING COUNT(*) > 0`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: weekly usage summaries: %w", err)
	}
	defer rows.Close()

	var out []WeeklyUsageSummary
	for rows.Next() {
		var w WeeklyUsageSummary
		if err := rows.Scan(&w.UserID, &w.RequestCount, &w.TotalTokens, &w.TotalCost); err != nil {
			return nil, fmt.Errorf("store: scan weekly usage summary: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// GetSilentUsers returns IDs of active-status users whose last_active_at falls
// within [minDaysAgo, maxDaysAgo) days ago. Used for layered winback campaigns.
// Users with NULL last_active_at (never recorded since the feature shipped) are
// excluded to avoid spamming brand-new or pre-feature accounts.
func (s *PgStore) GetSilentUsers(minDaysAgo, maxDaysAgo int) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT id FROM users
		 WHERE status = 'active'
		   AND last_active_at IS NOT NULL
		   AND last_active_at < NOW() - $1::int * INTERVAL '1 day'
		   AND last_active_at >= NOW() - $2::int * INTERVAL '1 day'`,
		minDaysAgo, maxDaysAgo,
	)
	if err != nil {
		return nil, fmt.Errorf("store: silent users: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: scan silent user: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
