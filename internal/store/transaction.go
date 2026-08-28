package store

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

// TokenUsage holds per-request token breakdown for billing transparency.
type TokenUsage struct {
	PromptTokens          int
	CompletionTokens      int
	CacheReadTokens       int
	CacheCreationTokens   int
	CacheCreation5mTokens int
	CacheCreation1hTokens int
}

// Transaction represents a billing transaction record.
type Transaction struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	Type                  string    `json:"type"`
	Amount                float64   `json:"amount"`
	BalanceAfter          float64   `json:"balance_after"`
	Model                 *string   `json:"model,omitempty"`
	Description           *string   `json:"description,omitempty"`
	RequestID             *string   `json:"request_id,omitempty"`
	PromptTokens          *int      `json:"prompt_tokens,omitempty"`
	CompletionTokens      *int      `json:"completion_tokens,omitempty"`
	CacheReadTokens       *int      `json:"cache_read_tokens,omitempty"`
	CacheCreationTokens   *int      `json:"cache_creation_tokens,omitempty"`
	CacheCreation5mTokens *int      `json:"cache_creation_5m_tokens,omitempty"`
	CacheCreation1hTokens *int      `json:"cache_creation_1h_tokens,omitempty"`
	SubscriptionID        *string   `json:"subscription_id,omitempty"`
	SubUserID             *string   `json:"sub_user_id,omitempty"`
	SubUserUsername       *string   `json:"sub_user_username,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

// DailyCost represents the cost for a single day.
type DailyCost struct {
	Date string  `json:"date"`
	Cost float64 `json:"cost"`
}

// ModelCost represents the cost breakdown for a single model.
type ModelCost struct {
	Model string  `json:"model"`
	Cost  float64 `json:"cost"`
}

// ModelSuccessStats holds per-model request success/failure counts and rate.
type ModelSuccessStats struct {
	Model        string  `json:"model"`
	SuccessCount int64   `json:"success_count"`
	FailureCount int64   `json:"failure_count"`
	TotalCount   int64   `json:"total_count"`
	SuccessRate  float64 `json:"success_rate"` // 0–100
}

// BillingStats holds billing statistics for a user.
type BillingStats struct {
	TodayCost      float64     `json:"today_cost"`
	MonthCost      float64     `json:"month_cost"`
	DailyTrend     []DailyCost `json:"daily_trend"`
	ModelBreakdown []ModelCost `json:"model_breakdown"`
}

// TenantBillingStats holds comprehensive billing statistics for a tenant.
type TenantBillingStats struct {
	TodayCost      float64          `json:"today_cost"`
	MonthCost      float64          `json:"month_cost"`
	DailyAverage   float64          `json:"daily_average"`
	DailyTrend     []DailyCost      `json:"daily_trend"`
	ModelBreakdown []ModelCost      `json:"model_breakdown"`
	SubUserRanking []SubUserCost    `json:"sub_user_ranking"`
	TokenStats     TenantTokenStats `json:"token_stats"`
}

// TenantTokenStats holds aggregated token counts for a tenant.
type TenantTokenStats struct {
	TotalPrompt        int64 `json:"total_prompt"`
	TotalCompletion    int64 `json:"total_completion"`
	TotalCacheRead     int64 `json:"total_cache_read"`
	TotalCacheCreation int64 `json:"total_cache_creation"`
}

// UserTokenStats holds aggregated token counts for a user.
type UserTokenStats struct {
	Prompt        int64 `json:"prompt"`
	Completion    int64 `json:"completion"`
	CacheRead     int64 `json:"cache_read"`
	CacheCreation int64 `json:"cache_creation"`
}

// UserTokenStatsPeriod is UserTokenStats with the days window it covers.
type UserTokenStatsPeriod struct {
	Days int `json:"days"`
	UserTokenStats
}

// UserTokenStatsResponse holds all-time and period-windowed token aggregates.
type UserTokenStatsResponse struct {
	AllTime UserTokenStats       `json:"all_time"`
	Period  UserTokenStatsPeriod `json:"period"`
}

// SubUserCost holds per-sub-user cost summary.
type SubUserCost struct {
	SubUserID       string  `json:"sub_user_id"`
	SubUserUsername string  `json:"sub_user_username"`
	TotalCost       float64 `json:"total_cost"`
	RequestCount    int64   `json:"request_count"`
}

// SubUserModelCost holds per-model cost breakdown for a sub-user.
type SubUserModelCost struct {
	Model               string  `json:"model"`
	Cost                float64 `json:"cost"`
	RequestCount        int64   `json:"request_count"`
	PromptTokens        int64   `json:"prompt_tokens"`
	CompletionTokens    int64   `json:"completion_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
}

// SubUserModelStats holds comprehensive model-level statistics for a sub-user.
type SubUserModelStats struct {
	SubUserID       string                 `json:"sub_user_id"`
	SubUserUsername string                 `json:"sub_user_username"`
	Period          *StatsPeriod           `json:"period,omitempty"`
	TotalCost       float64                `json:"total_cost"`
	TotalRequests   int64                  `json:"total_requests"`
	ModelBreakdown  []SubUserModelCost     `json:"model_breakdown"`
}

// StatsPeriod represents a date range for statistics.
type StatsPeriod struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// TransactionSums holds aggregated sums by transaction type.
type TransactionSums struct {
	TotalConsumption          float64 `json:"total_consumption"`
	TotalRecharge             float64 `json:"total_recharge"`
	TotalSubscriptionUsage    float64 `json:"total_subscription_usage"`
	TotalSubscriptionPurchase float64 `json:"total_sub_purchase"`
}

// APIKeyUsageSummary holds per-key usage statistics.
type APIKeyUsageSummary struct {
	KeyID        string  `json:"key_id"`
	KeyName      string  `json:"key_name"`
	KeyPrefix    string  `json:"key_prefix"`
	TotalCost    float64 `json:"total_cost"`
	RequestCount int64   `json:"request_count"`
	IsDeleted    bool    `json:"is_deleted"`
}

// ListTransactions returns paginated transactions for a user with optional type and date filters.
// Stats sums are computed without the type filter so the summary remains stable across tabs.
func (s *PgStore) ListTransactions(userID string, limit, offset int, typeFilter string, startDate, endDate *time.Time) ([]Transaction, int, *TransactionSums, error) {
	sums := &TransactionSums{}
	var total int

	// baseWhere: user_id + optional date range (used for sums, no type filter)
	baseWhere := "WHERE user_id = $1"
	baseArgs := []any{userID}
	argIdx := 2

	if startDate != nil {
		baseWhere += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		baseArgs = append(baseArgs, *startDate)
		argIdx++
	}
	if endDate != nil {
		baseWhere += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		baseArgs = append(baseArgs, *endDate)
		argIdx++
	}

	// sumQuery uses baseWhere (no type filter — sums reflect all types)
	sumQuery := fmt.Sprintf(
		`SELECT COALESCE(SUM(CASE WHEN type = 'consumption'        THEN amount ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN type = 'recharge'           THEN amount ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN type = 'subscription_usage' THEN amount ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN type = 'sub_purchase'       THEN amount ELSE 0 END), 0)
		 FROM transactions %s`, baseWhere)
	err := s.db.QueryRow(sumQuery, baseArgs...).Scan(&sums.TotalConsumption, &sums.TotalRecharge, &sums.TotalSubscriptionUsage, &sums.TotalSubscriptionPurchase)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("store: sum transactions: %w", err)
	}

	// listWhere: extends baseWhere with optional type filter
	listWhere := baseWhere
	listArgs := make([]any, len(baseArgs))
	copy(listArgs, baseArgs)
	listArgIdx := argIdx

	if typeFilter != "" {
		// Support comma-separated types for filtering (e.g., "checkin,task_reward")
		types := strings.Split(typeFilter, ",")
		if len(types) == 1 {
			listWhere += fmt.Sprintf(" AND type = $%d", listArgIdx)
			listArgs = append(listArgs, types[0])
			listArgIdx++
		} else {
			placeholders := make([]string, len(types))
			for i, t := range types {
				placeholders[i] = fmt.Sprintf("$%d", listArgIdx)
				listArgs = append(listArgs, strings.TrimSpace(t))
				listArgIdx++
			}
			listWhere += fmt.Sprintf(" AND type IN (%s)", strings.Join(placeholders, ","))
		}
	}

	// count query (with type filter, for pagination)
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM transactions %s`, listWhere)
	if err := s.db.QueryRow(countQuery, listArgs...).Scan(&total); err != nil {
		return nil, 0, nil, fmt.Errorf("store: count transactions: %w", err)
	}

	// list query
	listQuery := fmt.Sprintf(
		`SELECT id, user_id, type, amount, balance_after, model, description, request_id,
		        prompt_tokens, completion_tokens, cache_read_tokens,
		        cache_creation_tokens, cache_creation_5m_tokens, cache_creation_1h_tokens,
		        subscription_id, created_at
		 FROM transactions %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		listWhere, listArgIdx, listArgIdx+1)
	listArgs = append(listArgs, limit, offset)

	rows, err := s.db.Query(listQuery, listArgs...)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("store: list transactions: %w", err)
	}
	defer rows.Close()

	var txns []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.BalanceAfter,
			&t.Model, &t.Description, &t.RequestID,
			&t.PromptTokens, &t.CompletionTokens, &t.CacheReadTokens,
			&t.CacheCreationTokens, &t.CacheCreation5mTokens, &t.CacheCreation1hTokens,
			&t.SubscriptionID, &t.CreatedAt); err != nil {
			return nil, 0, nil, fmt.Errorf("store: scan transaction: %w", err)
		}
		txns = append(txns, t)
	}
	if txns == nil {
		txns = []Transaction{}
	}
	return txns, total, sums, nil
}

// UserConsumptionStats is the per-user counterpart of AdminConsumptionStats:
// per-model token/cost breakdown plus a daily cost trend, scoped to one user.
type UserConsumptionStats = AdminConsumptionStats

// GetUserConsumptionStats returns per-model consumption (request count, tokens, cost)
// and a daily cost trend for a single user over the specified number of days.
// Mirrors GetAdminConsumptionStats but filters every query by user_id.
func (s *PgStore) GetUserConsumptionStats(userID string, days int) (*UserConsumptionStats, error) {
	stats := &UserConsumptionStats{}

	g := new(errgroup.Group)

	// Query 1: per-model token and cost breakdown (scoped to this user).
	g.Go(func() error {
		rows, err := s.db.Query(
			`WITH model_names AS (
				SELECT DISTINCT model FROM transactions
				WHERE type IN ('consumption', 'subscription_usage')
				  AND user_id = $1
				  AND created_at >= CURRENT_DATE - $2::int * INTERVAL '1 day'
			),
			pricing_lookup AS (
				SELECT mn.model AS txn_model,
				       p.input_price, p.output_price, p.cached_input_price, p.cache_creation_price
				FROM model_names mn
				JOIN model_pricing p ON p.model_name = mn.model
				UNION ALL
				SELECT mn.model,
				       p.input_price, p.output_price, p.cached_input_price, p.cache_creation_price
				FROM model_names mn
				JOIN model_pricing p
				  ON position('/' in mn.model) > 0
				 AND split_part(p.model_name, '/', 2) = split_part(mn.model, '/', 2)
				WHERE NOT EXISTS (SELECT 1 FROM model_pricing WHERE model_name = mn.model)
				  AND (SELECT COUNT(DISTINCT mp.model_name) FROM model_pricing mp
				       WHERE split_part(mp.model_name, '/', 2) = split_part(mn.model, '/', 2)) = 1
			)
			SELECT
				COALESCE(t.model, 'unknown') AS model,
				COALESCE(SUM(t.prompt_tokens), 0),
				COALESCE(SUM(t.completion_tokens), 0),
				COALESCE(SUM(t.cache_read_tokens), 0),
				COALESCE(SUM(t.cache_creation_tokens), 0),
				COALESCE(SUM(t.amount), 0),
				COALESCE(SUM(t.prompt_tokens) * MAX(pl.input_price) / 1000000.0, 0),
				COALESCE(SUM(t.completion_tokens) * MAX(pl.output_price) / 1000000.0, 0),
				COALESCE(SUM(t.cache_read_tokens) * MAX(pl.cached_input_price) / 1000000.0, 0),
				COALESCE(SUM(t.cache_creation_tokens) * MAX(pl.cache_creation_price) / 1000000.0, 0),
				COUNT(*)
			 FROM transactions t
			 LEFT JOIN pricing_lookup pl ON pl.txn_model = t.model
			 WHERE t.type IN ('consumption', 'subscription_usage')
			   AND t.user_id = $1
			   AND t.created_at >= CURRENT_DATE - $2::int * INTERVAL '1 day'
			 GROUP BY t.model
			 ORDER BY SUM(t.amount) DESC`,
			userID, days,
		)
		if err != nil {
			return fmt.Errorf("store: user consumption stats models: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var m ModelTokenStats
			if err := rows.Scan(
				&m.Model,
				&m.PromptTokens, &m.CompletionTokens,
				&m.CacheReadTokens, &m.CacheCreationTokens,
				&m.TotalCost,
				&m.PromptCost, &m.CompletionCost,
				&m.CacheReadCost, &m.CacheCreationCost,
				&m.RequestCount,
			); err != nil {
				return fmt.Errorf("store: scan user model token stats: %w", err)
			}
			m.BreakdownEstimated = true
			stats.Models = append(stats.Models, m)
		}
		if stats.Models == nil {
			stats.Models = []ModelTokenStats{}
		}
		return nil
	})

	// Query 2: daily cost trend (scoped to this user).
	g.Go(func() error {
		rows, err := s.db.Query(
			`SELECT d::date AS date, COALESCE(SUM(t.amount), 0) AS cost
			 FROM generate_series(CURRENT_DATE - $2::int * INTERVAL '1 day', CURRENT_DATE, '1 day') d
			 LEFT JOIN transactions t ON t.user_id = $1 AND t.type IN ('consumption', 'subscription_usage')
			   AND t.created_at::date = d::date
			 GROUP BY d::date ORDER BY d::date`,
			userID, days,
		)
		if err != nil {
			return fmt.Errorf("store: user consumption daily trend: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var dc DailyCost
			var d time.Time
			if err := rows.Scan(&d, &dc.Cost); err != nil {
				return fmt.Errorf("store: scan user daily cost: %w", err)
			}
			dc.Date = d.Format("2006-01-02")
			stats.DailyTrend = append(stats.DailyTrend, dc)
		}
		if stats.DailyTrend == nil {
			stats.DailyTrend = []DailyCost{}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	for _, m := range stats.Models {
		stats.TotalCost += m.TotalCost
		stats.TotalRequests += m.RequestCount
	}

	return stats, nil
}

// ListUserTransactionsForExport returns up to 10000 transactions for a single user
// within an optional date range, ordered newest first, for Excel export.
func (s *PgStore) ListUserTransactionsForExport(userID string, startDate, endDate *time.Time) ([]Transaction, error) {
	query := `SELECT id, user_id, type, amount, balance_after, model, description, request_id, created_at,
		       prompt_tokens, completion_tokens, cache_read_tokens, cache_creation_tokens
		 FROM transactions
		 WHERE user_id = $1`
	args := []interface{}{userID}
	argIdx := 2

	if startDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *startDate)
		argIdx++
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *endDate)
		argIdx++
	}

	query += " ORDER BY created_at DESC LIMIT 10000"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list user transactions for export: %w", err)
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.BalanceAfter, &t.Model, &t.Description, &t.RequestID, &t.CreatedAt,
			&t.PromptTokens, &t.CompletionTokens, &t.CacheReadTokens, &t.CacheCreationTokens); err != nil {
			return nil, fmt.Errorf("store: scan user export transaction: %w", err)
		}
		transactions = append(transactions, t)
	}
	if transactions == nil {
		transactions = []Transaction{}
	}
	return transactions, rows.Err()
}

// GetBillingStats returns billing statistics for the given user over the specified number of days.
func (s *PgStore) GetBillingStats(userID string, days int) (*BillingStats, error) {
	stats := &BillingStats{}

	// Combined today + month cost in a single query.
	err := s.db.QueryRow(
		`SELECT
			COALESCE(SUM(CASE WHEN created_at >= CURRENT_DATE THEN amount END), 0),
			COALESCE(SUM(amount), 0)
		 FROM transactions
		 WHERE user_id = $1 AND type IN ('consumption', 'subscription_usage')
		 AND created_at >= date_trunc('month', CURRENT_DATE)`,
		userID,
	).Scan(&stats.TodayCost, &stats.MonthCost)
	if err != nil {
		return nil, fmt.Errorf("store: billing stats costs: %w", err)
	}

	// Run daily trend and model breakdown concurrently.
	g := new(errgroup.Group)

	g.Go(func() error {
		rows, err := s.db.Query(
			`SELECT d::date AS date, COALESCE(SUM(t.amount), 0) AS cost
			 FROM generate_series(CURRENT_DATE - $2::int * INTERVAL '1 day', CURRENT_DATE, '1 day') d
			 LEFT JOIN transactions t ON t.user_id = $1 AND t.type IN ('consumption', 'subscription_usage')
			   AND t.created_at::date = d::date
			 GROUP BY d::date ORDER BY d::date`,
			userID, days,
		)
		if err != nil {
			return fmt.Errorf("store: daily trend: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var dc DailyCost
			var d time.Time
			if err := rows.Scan(&d, &dc.Cost); err != nil {
				return fmt.Errorf("store: scan daily cost: %w", err)
			}
			dc.Date = d.Format("2006-01-02")
			stats.DailyTrend = append(stats.DailyTrend, dc)
		}
		if stats.DailyTrend == nil {
			stats.DailyTrend = []DailyCost{}
		}
		return nil
	})

	g.Go(func() error {
		rows, err := s.db.Query(
			`SELECT model, SUM(amount) FROM transactions
			 WHERE user_id = $1 AND type IN ('consumption', 'subscription_usage')
			 AND model IS NOT NULL
			 AND created_at >= CURRENT_DATE - $2::int * INTERVAL '1 day'
			 GROUP BY model ORDER BY SUM(amount) DESC`,
			userID, days,
		)
		if err != nil {
			return fmt.Errorf("store: model breakdown: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var mc ModelCost
			if err := rows.Scan(&mc.Model, &mc.Cost); err != nil {
				return fmt.Errorf("store: scan model cost: %w", err)
			}
			stats.ModelBreakdown = append(stats.ModelBreakdown, mc)
		}
		if stats.ModelBreakdown == nil {
			stats.ModelBreakdown = []ModelCost{}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetUserTokenStats returns all-time and period-windowed token aggregates for a user.
func (s *PgStore) GetUserTokenStats(userID string, days int) (*UserTokenStatsResponse, error) {
	resp := &UserTokenStatsResponse{Period: UserTokenStatsPeriod{Days: days}}

	g := new(errgroup.Group)

	g.Go(func() error {
		return s.db.QueryRow(
			`SELECT COALESCE(SUM(prompt_tokens), 0),
			        COALESCE(SUM(completion_tokens), 0),
			        COALESCE(SUM(cache_read_tokens), 0),
			        COALESCE(SUM(cache_creation_tokens), 0)
			 FROM transactions
			 WHERE user_id = $1
			   AND type IN ('consumption', 'subscription_usage')`,
			userID,
		).Scan(&resp.AllTime.Prompt, &resp.AllTime.Completion,
			&resp.AllTime.CacheRead, &resp.AllTime.CacheCreation)
	})

	g.Go(func() error {
		return s.db.QueryRow(
			`SELECT COALESCE(SUM(prompt_tokens), 0),
			        COALESCE(SUM(completion_tokens), 0),
			        COALESCE(SUM(cache_read_tokens), 0),
			        COALESCE(SUM(cache_creation_tokens), 0)
			 FROM transactions
			 WHERE user_id = $1
			   AND type IN ('consumption', 'subscription_usage')
			   AND created_at >= CURRENT_DATE - $2::int * INTERVAL '1 day'`,
			userID, days,
		).Scan(&resp.Period.Prompt, &resp.Period.Completion,
			&resp.Period.CacheRead, &resp.Period.CacheCreation)
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("store: user token stats: %w", err)
	}
	return resp, nil
}

// GetSubUserBillingStats returns billing statistics for a tenant sub-user.
func (s *PgStore) GetSubUserBillingStats(subUserID string, days int) (*BillingStats, error) {
	stats := &BillingStats{}

	// Get today and month cost from tenant_transactions
	err := s.db.QueryRow(
		`SELECT
			COALESCE(SUM(CASE WHEN created_at >= CURRENT_DATE THEN amount END), 0),
			COALESCE(SUM(amount), 0)
		 FROM tenant_transactions
		 WHERE sub_user_id = $1 AND type = 'consumption'
		 AND created_at >= date_trunc('month', CURRENT_DATE)`,
		subUserID,
	).Scan(&stats.TodayCost, &stats.MonthCost)
	if err != nil {
		return nil, fmt.Errorf("store: sub-user billing stats costs: %w", err)
	}

	// Daily trend and model breakdown
	g := new(errgroup.Group)

	g.Go(func() error {
		rows, err := s.db.Query(
			`SELECT d::date AS date, COALESCE(SUM(t.amount), 0) AS cost
			 FROM generate_series(CURRENT_DATE - $2::int * INTERVAL '1 day', CURRENT_DATE, '1 day') d
			 LEFT JOIN tenant_transactions t ON t.sub_user_id = $1 AND t.type = 'consumption'
			   AND t.created_at::date = d::date
			 GROUP BY d::date ORDER BY d::date`,
			subUserID, days,
		)
		if err != nil {
			return fmt.Errorf("store: sub-user daily trend: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var dc DailyCost
			var d time.Time
			if err := rows.Scan(&d, &dc.Cost); err != nil {
				return fmt.Errorf("store: scan sub-user daily cost: %w", err)
			}
			dc.Date = d.Format("2006-01-02")
			stats.DailyTrend = append(stats.DailyTrend, dc)
		}
		if stats.DailyTrend == nil {
			stats.DailyTrend = []DailyCost{}
		}
		return nil
	})

	g.Go(func() error {
		rows, err := s.db.Query(
			`SELECT model, COALESCE(SUM(amount), 0) AS cost
			 FROM tenant_transactions
			 WHERE sub_user_id = $1 AND type = 'consumption'
			   AND created_at >= CURRENT_DATE - $2::int * INTERVAL '1 day'
			 GROUP BY model ORDER BY cost DESC LIMIT 10`,
			subUserID, days,
		)
		if err != nil {
			return fmt.Errorf("store: sub-user model breakdown: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var mc ModelCost
			if err := rows.Scan(&mc.Model, &mc.Cost); err != nil {
				return fmt.Errorf("store: scan sub-user model cost: %w", err)
			}
			stats.ModelBreakdown = append(stats.ModelBreakdown, mc)
		}
		if stats.ModelBreakdown == nil {
			stats.ModelBreakdown = []ModelCost{}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return stats, nil
}

// RecordSubscriptionTransaction inserts a subscription_usage transaction.
// Balance is not modified; balance_after reflects the current balance at time of recording.
func (s *PgStore) RecordSubscriptionTransaction(userID, subscriptionID, model, requestID string, amount float64, tokens TokenUsage) error {
	var balanceAfter float64
	if err := s.db.QueryRow(
		`SELECT balance FROM balances WHERE user_id = $1`, userID,
	).Scan(&balanceAfter); err != nil {
		return fmt.Errorf("store: subscription tx get balance: %w", err)
	}

	_, err := s.db.Exec(
		`INSERT INTO transactions (user_id, type, amount, balance_after, model, request_id,
		        prompt_tokens, completion_tokens, cache_read_tokens,
		        cache_creation_tokens, cache_creation_5m_tokens, cache_creation_1h_tokens,
		        subscription_id)
		 VALUES ($1, 'subscription_usage', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		userID, amount, balanceAfter, model, requestID,
		intPtrOrNil(tokens.PromptTokens), intPtrOrNil(tokens.CompletionTokens),
		intPtrOrNil(tokens.CacheReadTokens), intPtrOrNil(tokens.CacheCreationTokens),
		intPtrOrNil(tokens.CacheCreation5mTokens), intPtrOrNil(tokens.CacheCreation1hTokens),
		subscriptionID,
	)
	if err != nil {
		return fmt.Errorf("store: insert subscription transaction: %w", err)
	}
	return nil
}

// GetAPIKeyUsageSummary returns per-key usage statistics for a user.
func (s *PgStore) GetAPIKeyUsageSummary(userID string, days int) ([]APIKeyUsageSummary, error) {
	rows, err := s.db.Query(
		`SELECT ak.id, ak.name, ak.key_prefix,
		        COALESCE(SUM(t.amount), 0) as total_cost,
		        COUNT(t.id) as request_count,
		        (ak.deleted_at IS NOT NULL) as is_deleted
		 FROM api_keys ak
		 LEFT JOIN transactions t ON t.api_key_id = ak.id
		   AND t.type IN ('consumption', 'subscription_usage')
		   AND t.created_at >= CASE
		       WHEN $2 = 0 THEN CURRENT_DATE
		       ELSE CURRENT_DATE - $2::int * INTERVAL '1 day'
		   END
		 WHERE ak.user_id = $1
		 GROUP BY ak.id, ak.name, ak.key_prefix, ak.deleted_at
		 ORDER BY total_cost DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get api key usage summary: %w", err)
	}
	defer rows.Close()

	var summaries []APIKeyUsageSummary
	for rows.Next() {
		var s APIKeyUsageSummary
		if err := rows.Scan(&s.KeyID, &s.KeyName, &s.KeyPrefix, &s.TotalCost, &s.RequestCount, &s.IsDeleted); err != nil {
			return nil, fmt.Errorf("store: scan api key usage: %w", err)
		}
		summaries = append(summaries, s)
	}
	if summaries == nil {
		summaries = []APIKeyUsageSummary{}
	}
	return summaries, nil
}

// RecordRequestFailure inserts a record for a failed model request (called asynchronously).
func (s *PgStore) RecordRequestFailure(userID, model string, httpStatus int) error {
	_, err := s.db.Exec(
		`INSERT INTO request_failures (user_id, model, http_status) VALUES ($1, $2, $3)`,
		userID, model, httpStatus,
	)
	return err
}

// GetModelSuccessStats returns per-model success/failure counts and success rate for a user.
func (s *PgStore) GetModelSuccessStats(userID string, days int) ([]ModelSuccessStats, error) {
	rows, err := s.db.Query(
		`SELECT model,
		    SUM(success_count) AS success_count,
		    SUM(failure_count) AS failure_count,
		    SUM(success_count) + SUM(failure_count) AS total_count,
		    CASE WHEN SUM(success_count) + SUM(failure_count) > 0
		         THEN ROUND(SUM(success_count)::numeric /
		              (SUM(success_count) + SUM(failure_count)) * 100, 2)
		         ELSE 100.0 END AS success_rate
		FROM (
		    SELECT model, COUNT(*) AS success_count, 0::bigint AS failure_count
		    FROM transactions
		    WHERE user_id = $1
		      AND model IS NOT NULL
		      AND created_at >= NOW() - ($2::int * INTERVAL '1 day')
		    GROUP BY model
		    UNION ALL
		    SELECT model, 0::bigint, COUNT(*)
		    FROM request_failures
		    WHERE user_id = $1
		      AND created_at >= NOW() - ($2::int * INTERVAL '1 day')
		    GROUP BY model
		) sub
		GROUP BY model
		ORDER BY total_count DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("store: model success stats: %w", err)
	}
	defer rows.Close()

	var result []ModelSuccessStats
	for rows.Next() {
		var m ModelSuccessStats
		if err := rows.Scan(&m.Model, &m.SuccessCount, &m.FailureCount, &m.TotalCount, &m.SuccessRate); err != nil {
			return nil, fmt.Errorf("store: scan model success stats: %w", err)
		}
		result = append(result, m)
	}
	if result == nil {
		result = []ModelSuccessStats{}
	}
	return result, nil
}

// GetAdminModelSuccessStats returns global per-model success/failure counts and success rate.
func (s *PgStore) GetAdminModelSuccessStats(days int) ([]ModelSuccessStats, error) {
	rows, err := s.db.Query(
		`SELECT model,
		    SUM(success_count) AS success_count,
		    SUM(failure_count) AS failure_count,
		    SUM(success_count) + SUM(failure_count) AS total_count,
		    CASE WHEN SUM(success_count) + SUM(failure_count) > 0
		         THEN ROUND(SUM(success_count)::numeric /
		              (SUM(success_count) + SUM(failure_count)) * 100, 2)
		         ELSE 100.0 END AS success_rate
		FROM (
		    SELECT model, COUNT(*) AS success_count, 0::bigint AS failure_count
		    FROM transactions
		    WHERE model IS NOT NULL
		      AND created_at >= NOW() - ($1::int * INTERVAL '1 day')
		    GROUP BY model
		    UNION ALL
		    SELECT model, 0::bigint, COUNT(*)
		    FROM request_failures
		    WHERE created_at >= NOW() - ($1::int * INTERVAL '1 day')
		    GROUP BY model
		) sub
		GROUP BY model
		ORDER BY total_count DESC`,
		days,
	)
	if err != nil {
		return nil, fmt.Errorf("store: admin model success stats: %w", err)
	}
	defer rows.Close()

	var result []ModelSuccessStats
	for rows.Next() {
		var m ModelSuccessStats
		if err := rows.Scan(&m.Model, &m.SuccessCount, &m.FailureCount, &m.TotalCount, &m.SuccessRate); err != nil {
			return nil, fmt.Errorf("store: scan admin model success stats: %w", err)
		}
		result = append(result, m)
	}
	if result == nil {
		result = []ModelSuccessStats{}
	}
	return result, nil
}

// ListTransactionsByAPIKey returns paginated transactions for a specific API key.
func (s *PgStore) ListTransactionsByAPIKey(keyID string, limit, offset int) ([]Transaction, int, error) {
	var total int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM transactions WHERE api_key_id = $1`,
		keyID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("store: count key transactions: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT id, user_id, type, amount, balance_after, model, description, request_id,
		        prompt_tokens, completion_tokens, cache_read_tokens,
		        cache_creation_tokens, cache_creation_5m_tokens, cache_creation_1h_tokens,
		        subscription_id, created_at
		 FROM transactions
		 WHERE api_key_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		keyID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list key transactions: %w", err)
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Type, &t.Amount, &t.BalanceAfter,
			&t.Model, &t.Description, &t.RequestID,
			&t.PromptTokens, &t.CompletionTokens, &t.CacheReadTokens,
			&t.CacheCreationTokens, &t.CacheCreation5mTokens, &t.CacheCreation1hTokens,
			&t.SubscriptionID, &t.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("store: scan key transaction: %w", err)
		}
		txs = append(txs, t)
	}
	if txs == nil {
		txs = []Transaction{}
	}
	return txs, total, nil
}

