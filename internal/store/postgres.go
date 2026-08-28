package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

// PgStore implements Store backed by PostgreSQL.
type PgStore struct {
	db *sql.DB
}

// compile-time check
var _ Store = (*PgStore)(nil)

// OpenPostgres opens a PostgreSQL connection using the given DSN and pool settings.
func OpenPostgres(dsn string, maxOpen, maxIdle int, maxLifetime time.Duration) (*PgStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open postgres: %w", err)
	}
	if maxOpen <= 0 {
		maxOpen = 50
	}
	if maxIdle <= 0 {
		maxIdle = 15
	}
	if maxLifetime <= 0 {
		maxLifetime = 5 * time.Minute
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: ping postgres: %w", err)
	}
	return &PgStore{db: db}, nil
}

// DB returns the underlying *sql.DB for use with golang-migrate.
func (s *PgStore) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *PgStore) Close() error {
	return s.db.Close()
}

// ---------- User methods ----------

func (s *PgStore) CreateUser(phone, password, role string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("store: hash password: %w", err)
	}

	var u User
	var emailVerified sql.NullBool
	var emailVerifiedAt sql.NullTime
	err = s.db.QueryRow(
		`INSERT INTO users (phone, password_hash, role)
		 VALUES ($1, $2, $3)
		 RETURNING id, COALESCE(phone,''), COALESCE(email,''), password_hash, COALESCE(nickname,''),
		           role, status, email_verified, email_verified_at,
		           first_recharge_bonus_granted, created_at, updated_at`,
		phone, string(hash), role,
	).Scan(&u.ID, &u.Phone, &u.Email, &u.PasswordHash, &u.Nickname, &u.Role, &u.Status,
		&emailVerified, &emailVerifiedAt, &u.FirstRechargeBonusGranted, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create user: %w", err)
	}
	u.EmailVerified = emailVerified.Bool
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (s *PgStore) CreateUserWithEmail(email, password, role string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("store: hash password: %w", err)
	}

	var u User
	var emailVerified sql.NullBool
	var emailVerifiedAt sql.NullTime
	err = s.db.QueryRow(
		`INSERT INTO users (email, password_hash, role, email_verified)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, COALESCE(phone,''), email, password_hash, COALESCE(nickname,''),
		           role, status, email_verified, email_verified_at,
		           first_recharge_bonus_granted, created_at, updated_at`,
		email, string(hash), role, false,
	).Scan(&u.ID, &u.Phone, &u.Email, &u.PasswordHash, &u.Nickname, &u.Role, &u.Status,
		&emailVerified, &emailVerifiedAt, &u.FirstRechargeBonusGranted, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create user with email: %w", err)
	}
	u.EmailVerified = emailVerified.Bool
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (s *PgStore) Authenticate(phone, password string) (*User, error) {
	var u User
	var emailVerified sql.NullBool
	var emailVerifiedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, COALESCE(phone,''), COALESCE(email,''), password_hash, COALESCE(nickname, ''),
		        role, status, email_verified, email_verified_at,
		        first_recharge_bonus_granted, created_at, updated_at
		 FROM users WHERE phone = $1`,
		phone,
	).Scan(&u.ID, &u.Phone, &u.Email, &u.PasswordHash, &u.Nickname, &u.Role, &u.Status,
		&emailVerified, &emailVerifiedAt, &u.FirstRechargeBonusGranted, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("store: query user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	u.EmailVerified = emailVerified.Bool
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (s *PgStore) AuthenticateByEmail(email, password string) (*User, error) {
	var u User
	var emailVerified sql.NullBool
	var emailVerifiedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, COALESCE(phone,''), email, password_hash, COALESCE(nickname, ''),
		        role, status, email_verified, email_verified_at,
		        first_recharge_bonus_granted, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Phone, &u.Email, &u.PasswordHash, &u.Nickname, &u.Role, &u.Status,
		&emailVerified, &emailVerifiedAt, &u.FirstRechargeBonusGranted, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("store: query user by email: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	u.EmailVerified = emailVerified.Bool
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (s *PgStore) AuthenticateByIdentifier(identifier, password string) (*User, error) {
	// 自动检测是邮箱还是手机号
	if isEmail(identifier) {
		return s.AuthenticateByEmail(identifier, password)
	}
	return s.Authenticate(identifier, password)
}

func (s *PgStore) UserCount() (int64, error) {
	var count int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: count users: %w", err)
	}
	return count, nil
}

func (s *PgStore) GetUserByID(id string) (*User, error) {
	var u User
	var emailVerified sql.NullBool
	var emailVerifiedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, COALESCE(phone,''), COALESCE(email,''), password_hash, COALESCE(nickname, ''),
		        role, status, email_verified, email_verified_at,
		        first_recharge_bonus_granted, image_share_enabled, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Phone, &u.Email, &u.PasswordHash, &u.Nickname, &u.Role, &u.Status,
		&emailVerified, &emailVerifiedAt, &u.FirstRechargeBonusGranted, &u.ImageShareEnabled, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user by id: %w", err)
	}
	u.EmailVerified = emailVerified.Bool
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (s *PgStore) GetUserByPhone(phone string) (*User, error) {
	var u User
	var emailVerified sql.NullBool
	var emailVerifiedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, phone, COALESCE(email,''), password_hash, COALESCE(nickname, ''),
		        role, status, email_verified, email_verified_at,
		        first_recharge_bonus_granted, image_share_enabled, created_at, updated_at
		 FROM users WHERE phone = $1`,
		phone,
	).Scan(&u.ID, &u.Phone, &u.Email, &u.PasswordHash, &u.Nickname, &u.Role, &u.Status,
		&emailVerified, &emailVerifiedAt, &u.FirstRechargeBonusGranted, &u.ImageShareEnabled, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user by phone: %w", err)
	}
	u.EmailVerified = emailVerified.Bool
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (s *PgStore) GetUserByEmail(email string) (*User, error) {
	var u User
	var emailVerified sql.NullBool
	var emailVerifiedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, COALESCE(phone,''), email, password_hash, COALESCE(nickname, ''),
		        role, status, email_verified, email_verified_at,
		        first_recharge_bonus_granted, image_share_enabled, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Phone, &u.Email, &u.PasswordHash, &u.Nickname, &u.Role, &u.Status,
		&emailVerified, &emailVerifiedAt, &u.FirstRechargeBonusGranted, &u.ImageShareEnabled, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user by email: %w", err)
	}
	u.EmailVerified = emailVerified.Bool
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (s *PgStore) GetUserByIdentifier(identifier string) (*User, error) {
	if isEmail(identifier) {
		return s.GetUserByEmail(identifier)
	}
	return s.GetUserByPhone(identifier)
}

func (s *PgStore) MarkFirstRechargeGranted(userID string) (bool, error) {
	res, err := s.db.Exec(
		`UPDATE users SET first_recharge_bonus_granted = TRUE, updated_at = NOW()
		 WHERE id = $1 AND first_recharge_bonus_granted = FALSE`,
		userID,
	)
	if err != nil {
		return false, fmt.Errorf("store: mark first recharge bonus: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *PgStore) MarkEmailVerified(userID string) error {
	_, err := s.db.Exec(
		`UPDATE users SET email_verified = TRUE, email_verified_at = NOW(), updated_at = NOW()
		 WHERE id = $1`,
		userID,
	)
	return err
}

func (s *PgStore) UpdatePassword(phone, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("store: hash password: %w", err)
	}
	res, err := s.db.Exec(
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE phone = $2`,
		string(hash), phone,
	)
	if err != nil {
		return fmt.Errorf("store: update password: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: user not found")
	}
	return nil
}

func (s *PgStore) UpdatePasswordByEmail(email, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("store: hash password: %w", err)
	}
	res, err := s.db.Exec(
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE email = $2`,
		string(hash), email,
	)
	if err != nil {
		return fmt.Errorf("store: update password by email: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: user not found")
	}
	return nil
}

func (s *PgStore) ListUsers(limit, offset int) ([]User, int, error) {
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("store: count users: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT id, phone, password_hash, COALESCE(nickname, ''), role, status,
		        first_recharge_bonus_granted, image_share_enabled, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Phone, &u.PasswordHash, &u.Nickname, &u.Role, &u.Status,
			&u.FirstRechargeBonusGranted, &u.ImageShareEnabled, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan user: %w", err)
		}
		users = append(users, u)
	}
	if users == nil {
		users = []User{}
	}
	return users, total, nil
}

func (s *PgStore) ListUsersWithBalance(limit, offset int, search string) (
	users []UserWithBalance, filteredTotal int,
	globalTotal int, globalActiveCount int, globalTotalBalance float64,
	err error,
) {
	// Global stats — never filtered by search, so stat cards stay stable while user types in the search box.
	if err = s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&globalTotal); err != nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("store: count users: %w", err)
	}
	if err = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE status = 'active'").Scan(&globalActiveCount); err != nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("store: count active users: %w", err)
	}
	if err = s.db.QueryRow("SELECT COALESCE(SUM(balance), 0) FROM balances").Scan(&globalTotalBalance); err != nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("store: sum balances: %w", err)
	}

	var where string
	var args []any
	if search != "" {
		where = " WHERE (u.phone LIKE '%' || $1 || '%' OR u.email LIKE '%' || $1 || '%')"
		args = append(args, search)
	}

	if search == "" {
		filteredTotal = globalTotal
	} else {
		if err = s.db.QueryRow("SELECT COUNT(*) FROM users u"+where, args...).Scan(&filteredTotal); err != nil {
			return nil, 0, 0, 0, 0, fmt.Errorf("store: count filtered users: %w", err)
		}
	}

	listQuery := `SELECT u.id, COALESCE(u.phone, ''), COALESCE(u.email, ''), u.password_hash, COALESCE(u.nickname, ''), u.role, u.status,
		        u.first_recharge_bonus_granted, u.image_share_enabled, u.created_at, u.updated_at,
		        COALESCE(b.balance, 0)
		 FROM users u
		 LEFT JOIN balances b ON b.user_id = u.id` + where +
		fmt.Sprintf(" ORDER BY u.created_at DESC LIMIT %d OFFSET %d", limit, offset)
	rows, err := s.db.Query(listQuery, args...)
	if err != nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("store: list users with balance: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ub UserWithBalance
		if err = rows.Scan(&ub.ID, &ub.Phone, &ub.Email, &ub.PasswordHash, &ub.Nickname, &ub.Role, &ub.Status,
			&ub.FirstRechargeBonusGranted, &ub.ImageShareEnabled, &ub.CreatedAt, &ub.UpdatedAt, &ub.Balance); err != nil {
			return nil, 0, 0, 0, 0, fmt.Errorf("store: scan user with balance: %w", err)
		}
		users = append(users, ub)
	}
	if users == nil {
		users = []UserWithBalance{}
	}
	return users, filteredTotal, globalTotal, globalActiveCount, globalTotalBalance, nil
}

func (s *PgStore) GetAdminDashboardStats() (*AdminDashboardStats, error) {
	var stats AdminDashboardStats

	// Total users
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	if err != nil {
		return nil, fmt.Errorf("store: count users: %w", err)
	}

	// Today's revenue from paid orders (excludes bonuses/trial credits)
	err = s.db.QueryRow(
		`SELECT COALESCE(SUM(amount), 0) FROM orders
		 WHERE status = 'paid' AND pay_time >= CURRENT_DATE`,
	).Scan(&stats.TodayRevenue)
	if err != nil {
		return nil, fmt.Errorf("store: today revenue: %w", err)
	}

	// Today's requests from consumption + subscription_usage transactions
	err = s.db.QueryRow(
		`SELECT COUNT(*) FROM transactions
		 WHERE type IN ('consumption', 'subscription_usage') AND created_at >= CURRENT_DATE`,
	).Scan(&stats.TodayRequests)
	if err != nil {
		return nil, fmt.Errorf("store: today requests: %w", err)
	}

	return &stats, nil
}

// GetAdminFunnelStats computes the acquisition→activation→retention funnel for
// the cohort of users registered within the window [now-days, now]:
//
//	registered  → users created in the window
//	first_recharge → those with ≥1 paid order
//	repeat_recharge → those with ≥2 paid orders
//	post_recharge_use → those with a consumption/subscription_usage txn dated
//	                    after their first paid order
//
// All counts are restricted to the same registration cohort so the rates are
// honest (later stages are always subsets of "registered"). Uses orders.status
// = 'paid' + pay_time as the source of truth for recharges.
func (s *PgStore) GetAdminFunnelStats(days int) (*AdminFunnelStats, error) {
	var registered, firstRecharge, repeatRecharge, postRechargeUse int64

	err := s.db.QueryRow(
		`WITH cohort AS (
			SELECT id FROM users
			WHERE created_at >= CURRENT_DATE - $1::int * INTERVAL '1 day'
		),
		first_paid AS (
			SELECT o.user_id, MIN(o.pay_time) AS first_pay_time, COUNT(*) AS paid_count
			FROM orders o
			JOIN cohort c ON c.id = o.user_id
			WHERE o.status = 'paid' AND o.pay_time IS NOT NULL
			GROUP BY o.user_id
		),
		post_use AS (
			SELECT DISTINCT fp.user_id
			FROM first_paid fp
			JOIN transactions t ON t.user_id = fp.user_id
			WHERE t.type IN ('consumption', 'subscription_usage')
			  AND t.created_at >= fp.first_pay_time
		)
		SELECT
			(SELECT COUNT(*) FROM cohort),
			(SELECT COUNT(*) FROM first_paid),
			(SELECT COUNT(*) FROM first_paid WHERE paid_count >= 2),
			(SELECT COUNT(*) FROM post_use)`,
		days,
	).Scan(&registered, &firstRecharge, &repeatRecharge, &postRechargeUse)
	if err != nil {
		return nil, fmt.Errorf("store: funnel stats: %w", err)
	}

	stats := &AdminFunnelStats{
		Days: int64(days),
		Stages: []FunnelStage{
			{Key: "registered", Label: "注册", Count: registered},
			{Key: "first_recharge", Label: "首充", Count: firstRecharge},
			{Key: "repeat_recharge", Label: "复充", Count: repeatRecharge},
			{Key: "post_recharge_use", Label: "首充后有消费", Count: postRechargeUse},
		},
	}
	if registered > 0 {
		stats.FirstRechargeRate = float64(firstRecharge) / float64(registered)
	}
	if firstRecharge > 0 {
		stats.RepeatRechargeRate = float64(repeatRecharge) / float64(firstRecharge)
		stats.PostRechargeUseRate = float64(postRechargeUse) / float64(firstRecharge)
	}
	return stats, nil
}

// GetAdminConsumptionStats returns global token consumption and cost breakdown per model.
func (s *PgStore) GetAdminConsumptionStats(days int) (*AdminConsumptionStats, error) {
	stats := &AdminConsumptionStats{}

	g := new(errgroup.Group)

	// Query 1: per-model token and cost breakdown.
	// Uses a CTE to build a pricing lookup that first tries exact match on model_name,
	// then falls back to suffix match (after '/') only when the suffix is unambiguous.
	g.Go(func() error {
		rows, err := s.db.Query(
			`WITH model_names AS (
				SELECT DISTINCT model FROM transactions
				WHERE type IN ('consumption', 'subscription_usage')
				  AND created_at >= CURRENT_DATE - $1::int * INTERVAL '1 day'
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
			   AND t.created_at >= CURRENT_DATE - $1::int * INTERVAL '1 day'
			 GROUP BY t.model
			 ORDER BY SUM(t.amount) DESC`,
			days,
		)
		if err != nil {
			return fmt.Errorf("store: consumption stats models: %w", err)
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
				return fmt.Errorf("store: scan model token stats: %w", err)
			}
			m.BreakdownEstimated = true
			stats.Models = append(stats.Models, m)
		}
		if stats.Models == nil {
			stats.Models = []ModelTokenStats{}
		}
		return nil
	})

	// Query 2: daily cost trend.
	g.Go(func() error {
		rows, err := s.db.Query(
			`SELECT d::date AS date, COALESCE(SUM(t.amount), 0) AS cost
			 FROM generate_series(CURRENT_DATE - $1::int * INTERVAL '1 day', CURRENT_DATE, '1 day') d
			 LEFT JOIN transactions t ON t.type IN ('consumption', 'subscription_usage')
			   AND t.created_at::date = d::date
			 GROUP BY d::date ORDER BY d::date`,
			days,
		)
		if err != nil {
			return fmt.Errorf("store: consumption daily trend: %w", err)
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

	var successStats []ModelSuccessStats
	g.Go(func() error {
		var err error
		successStats, err = s.GetAdminModelSuccessStats(days)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Compute totals from per-model results and merge success stats.
	ssMap := make(map[string]ModelSuccessStats, len(successStats))
	for _, ss := range successStats {
		ssMap[ss.Model] = ss
	}
	for i := range stats.Models {
		stats.TotalCost += stats.Models[i].TotalCost
		stats.TotalRequests += stats.Models[i].RequestCount
		if ss, ok := ssMap[stats.Models[i].Model]; ok {
			stats.Models[i].FailureCount = ss.FailureCount
			stats.Models[i].SuccessRate = ss.SuccessRate
		} else {
			stats.Models[i].SuccessRate = 100.0
		}
	}

	return stats, nil
}

// GetImageDurationStats returns per-model image generation duration stats (min/avg/max in seconds).
func (s *PgStore) GetImageDurationStats(days int) ([]ImageDurationStats, error) {
	rows, err := s.db.Query(
		`SELECT
			model,
			COUNT(*) AS request_count,
			ROUND(MIN(EXTRACT(EPOCH FROM (completed_at - started_at)))::numeric, 2) AS min_seconds,
			ROUND(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)))::numeric, 2) AS avg_seconds,
			ROUND(MAX(EXTRACT(EPOCH FROM (completed_at - started_at)))::numeric, 2) AS max_seconds
		FROM image_tasks
		WHERE status = 'completed'
		  AND started_at IS NOT NULL
		  AND completed_at IS NOT NULL
		  AND completed_at > started_at
		  AND created_at >= NOW() - ($1::int * INTERVAL '1 day')
		GROUP BY model
		ORDER BY avg_seconds DESC`,
		days,
	)
	if err != nil {
		return nil, fmt.Errorf("store: image duration stats: %w", err)
	}
	defer rows.Close()

	var result []ImageDurationStats
	for rows.Next() {
		var d ImageDurationStats
		if err := rows.Scan(&d.Model, &d.RequestCount, &d.MinSeconds, &d.AvgSeconds, &d.MaxSeconds); err != nil {
			return nil, fmt.Errorf("store: scan image duration stats: %w", err)
		}
		result = append(result, d)
	}
	if result == nil {
		result = []ImageDurationStats{}
	}
	return result, nil
}

func (s *PgStore) UpdateUserStatus(id, status string) error {
	res, err := s.db.Exec(
		"UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2",
		status, id,
	)
	if err != nil {
		return fmt.Errorf("store: update user status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: user not found")
	}
	return nil
}

// DeleteUser removes a user and all associated data in a single transaction.
func (s *PgStore) DeleteUser(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	var tenantOwned int
	if err := tx.QueryRow("SELECT COUNT(*) FROM tenants WHERE owner_id = $1", id).Scan(&tenantOwned); err != nil {
		return fmt.Errorf("store: count owned tenants: %w", err)
	}
	if tenantOwned > 0 {
		return fmt.Errorf("store: user owns tenants, transfer ownership before deletion")
	}

	if _, err := tx.Exec("DELETE FROM invoice_request_orders WHERE order_id IN (SELECT id FROM orders WHERE user_id = $1)", id); err != nil {
		return fmt.Errorf("store: delete invoice request orders: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM invoice_requests WHERE user_id = $1", id); err != nil {
		return fmt.Errorf("store: delete invoice requests: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM invoice_titles WHERE user_id = $1", id); err != nil {
		return fmt.Errorf("store: delete invoice titles: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM orders WHERE user_id = $1", id); err != nil {
		return fmt.Errorf("store: delete orders: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM transactions WHERE user_id = $1", id); err != nil {
		return fmt.Errorf("store: delete transactions: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM balances WHERE user_id = $1", id); err != nil {
		return fmt.Errorf("store: delete balance: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM subscription_usage WHERE user_id = $1", id); err != nil {
		return fmt.Errorf("store: delete subscription usage: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM user_subscriptions WHERE user_id = $1", id); err != nil {
		return fmt.Errorf("store: delete user subscriptions: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM subscription_orders WHERE user_id = $1", id); err != nil {
		return fmt.Errorf("store: delete subscription orders: %w", err)
	}
	if _, err := tx.Exec("UPDATE tenant_transactions SET operator_id = NULL WHERE operator_id = $1", id); err != nil {
		return fmt.Errorf("store: clear tenant transactions operator: %w", err)
	}
	if _, err := tx.Exec("UPDATE tenants SET created_by_admin = NULL WHERE created_by_admin = $1", id); err != nil {
		return fmt.Errorf("store: clear tenants created_by_admin: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM tenant_invitations WHERE invited_by = $1", id); err != nil {
		return fmt.Errorf("store: delete tenant invitations: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM tenant_api_keys WHERE created_by = $1", id); err != nil {
		return fmt.Errorf("store: delete tenant api keys: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM tenant_sub_users WHERE created_by = $1", id); err != nil {
		return fmt.Errorf("store: delete tenant sub users: %w", err)
	}
	// api_keys, chat_sessions/messages cascade; referred_by set null automatically.
	res, err := tx.Exec("DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("store: delete user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: user not found")
	}

	return tx.Commit()
}

// ---------- Model methods ----------

func (s *PgStore) ListModels() ([]Model, error) {
	rows, err := s.db.Query("SELECT id, name, COALESCE(display_name,''), COALESCE(category,''), created_at, updated_at FROM models ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("store: list models: %w", err)
	}
	defer rows.Close()

	var models []Model
	for rows.Next() {
		var m Model
		if err := rows.Scan(&m.ID, &m.Name, &m.DisplayName, &m.Category, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan model: %w", err)
		}
		models = append(models, m)
	}

	for i := range models {
		upstreams, err := s.listUpstreams(models[i].ID)
		if err != nil {
			return nil, err
		}
		models[i].Upstreams = upstreams
	}
	if models == nil {
		models = []Model{}
	}
	return models, nil
}

func (s *PgStore) GetModel(id int64) (*Model, error) {
	var m Model
	err := s.db.QueryRow(
		"SELECT id, name, COALESCE(display_name,''), COALESCE(category,''), created_at, updated_at FROM models WHERE id = $1", id,
	).Scan(&m.ID, &m.Name, &m.DisplayName, &m.Category, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: get model %d: %w", id, err)
	}
	upstreams, err := s.listUpstreams(m.ID)
	if err != nil {
		return nil, err
	}
	m.Upstreams = upstreams
	return &m, nil
}

func (s *PgStore) GetModelByName(name string) (*Model, error) {
	var m Model
	err := s.db.QueryRow(
		"SELECT id, name, COALESCE(display_name,''), COALESCE(category,''), created_at, updated_at FROM models WHERE name = $1", name,
	).Scan(&m.ID, &m.Name, &m.DisplayName, &m.Category, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: model not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("store: get model by name: %w", err)
	}
	upstreams, err := s.listUpstreams(m.ID)
	if err != nil {
		return nil, err
	}
	m.Upstreams = upstreams
	return &m, nil
}

func (s *PgStore) CreateModel(name, displayName, category string, upstreams []Upstream) (*Model, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	var modelID int64
	err = tx.QueryRow(
		"INSERT INTO models (name, display_name, category) VALUES ($1, $2, $3) RETURNING id",
		name, displayName, category,
	).Scan(&modelID)
	if err != nil {
		return nil, fmt.Errorf("store: insert model: %w", err)
	}

	for i, u := range upstreams {
		_, err := tx.Exec(
			`INSERT INTO upstreams (model_id, provider, protocol, protocols, upstream_provider, upstream_name, base_url, api_key, model_override, weight, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			modelID, u.Provider, u.Protocol, pq.Array(u.Protocols), u.UpstreamProvider, u.UpstreamName, u.BaseURL, u.APIKey, u.ModelOverride, u.Weight, i,
		)
		if err != nil {
			return nil, fmt.Errorf("store: insert upstream: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit: %w", err)
	}
	return s.GetModel(modelID)
}

func (s *PgStore) UpdateModel(id int64, name, displayName, category string, upstreams []Upstream) (*Model, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE models SET name = $1, display_name = $2, category = $3, updated_at = NOW() WHERE id = $4", name, displayName, category, id)
	if err != nil {
		return nil, fmt.Errorf("store: update model: %w", err)
	}

	_, err = tx.Exec("DELETE FROM upstreams WHERE model_id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("store: delete upstreams: %w", err)
	}

	for i, u := range upstreams {
		_, err := tx.Exec(
			`INSERT INTO upstreams (model_id, provider, protocol, protocols, upstream_provider, upstream_name, base_url, api_key, model_override, weight, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			id, u.Provider, u.Protocol, pq.Array(u.Protocols), u.UpstreamProvider, u.UpstreamName, u.BaseURL, u.APIKey, u.ModelOverride, u.Weight, i,
		)
		if err != nil {
			return nil, fmt.Errorf("store: insert upstream: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit: %w", err)
	}
	return s.GetModel(id)
}

func (s *PgStore) DeleteModel(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	var name string
	if err := tx.QueryRow("SELECT name FROM models WHERE id = $1", id).Scan(&name); err != nil {
		return fmt.Errorf("store: get model name: %w", err)
	}

	if _, err := tx.Exec("DELETE FROM model_pricing WHERE model_name = $1", name); err != nil {
		return fmt.Errorf("store: delete model pricing: %w", err)
	}

	if _, err := tx.Exec("DELETE FROM models WHERE id = $1", id); err != nil {
		return fmt.Errorf("store: delete model: %w", err)
	}

	return tx.Commit()
}

func (s *PgStore) listUpstreams(modelID int64) ([]Upstream, error) {
	rows, err := s.db.Query(
		`SELECT id, model_id, provider, COALESCE(protocol,''), COALESCE(protocols, ARRAY[]::text[]), COALESCE(upstream_provider,''), COALESCE(upstream_name,''), base_url, api_key, model_override, weight, sort_order
		 FROM upstreams WHERE model_id = $1 ORDER BY sort_order`,
		modelID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list upstreams: %w", err)
	}
	defer rows.Close()

	var upstreams []Upstream
	for rows.Next() {
		var u Upstream
		if err := rows.Scan(&u.ID, &u.ModelID, &u.Provider, &u.Protocol, pq.Array(&u.Protocols), &u.UpstreamProvider, &u.UpstreamName, &u.BaseURL, &u.APIKey, &u.ModelOverride, &u.Weight, &u.SortOrder); err != nil {
			return nil, fmt.Errorf("store: scan upstream: %w", err)
		}
		upstreams = append(upstreams, u)
	}
	if upstreams == nil {
		upstreams = []Upstream{}
	}
	return upstreams, nil
}

// ---------- API Key methods ----------

func (s *PgStore) CreateAPIKey(userID, keyHash, keyPrefix, name string, planID *int) (*APIKey, error) {
	var k APIKey
	err := s.db.QueryRow(
		`INSERT INTO api_keys (user_id, key_hash, key_prefix, name, plan_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, key_hash, key_prefix, name, status, plan_id, last_used_at, created_at`,
		userID, keyHash, keyPrefix, name, planID,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Status, &k.PlanID, &k.LastUsedAt, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create api key: %w", err)
	}
	return &k, nil
}

func (s *PgStore) ListAPIKeysByUser(userID string) ([]APIKey, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, key_hash, key_prefix, name, status, plan_id, last_used_at, created_at
		 FROM api_keys WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list api keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Status, &k.PlanID, &k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []APIKey{}
	}
	return keys, nil
}

func (s *PgStore) GetAPIKeyByID(id string) (*APIKey, error) {
	var k APIKey
	err := s.db.QueryRow(
		`SELECT id, user_id, key_hash, key_prefix, name, status, plan_id, last_used_at, created_at
		 FROM api_keys WHERE id = $1`,
		id,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Status, &k.PlanID, &k.LastUsedAt, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: api key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get api key by id: %w", err)
	}
	return &k, nil
}

func (s *PgStore) GetActiveAPIKeyByIDAndUser(keyID, userID string) (*APIKey, error) {
	var k APIKey
	err := s.db.QueryRow(
		`SELECT id, user_id, key_hash, key_prefix, name, status, plan_id, last_used_at, created_at
		 FROM api_keys WHERE id = $1 AND user_id = $2 AND status = 'active' AND deleted_at IS NULL`,
		keyID, userID,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Status, &k.PlanID, &k.LastUsedAt, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get active api key by id and user: %w", err)
	}
	return &k, nil
}


func (s *PgStore) GetAPIKeyByHash(keyHash string) (*APIKey, error) {
	var k APIKey
	err := s.db.QueryRow(
		`SELECT id, user_id, key_hash, key_prefix, name, status, plan_id, last_used_at, created_at
		 FROM api_keys WHERE key_hash = $1 AND status = 'active' AND deleted_at IS NULL`,
		keyHash,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Status, &k.PlanID, &k.LastUsedAt, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: api key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get api key: %w", err)
	}
	return &k, nil
}

func (s *PgStore) DeleteAPIKey(id, userID string) error {
	res, err := s.db.Exec(
		"UPDATE api_keys SET deleted_at = NOW(), status = 'deleted' WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL",
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("store: delete api key: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: api key not found")
	}
	return nil
}

// RevokeAllAPIKeys soft-deletes all API keys for a user and returns the count deleted.
func (s *PgStore) RevokeAllAPIKeys(userID string) (int, error) {
	res, err := s.db.Exec("UPDATE api_keys SET deleted_at = NOW(), status = 'deleted' WHERE user_id = $1 AND deleted_at IS NULL", userID)
	if err != nil {
		return 0, fmt.Errorf("store: revoke all api keys: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *PgStore) TouchAPIKeyLastUsed(id string) error {
	_, err := s.db.Exec("UPDATE api_keys SET last_used_at = NOW() WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("store: touch api key: %w", err)
	}
	return nil
}

func (s *PgStore) BatchTouchAPIKeysLastUsed(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.db.Exec("UPDATE api_keys SET last_used_at = NOW() WHERE id = ANY($1)", ids)
	if err != nil {
		return fmt.Errorf("store: batch touch api keys: %w", err)
	}
	return nil
}

// BatchTouchUsersLastActive updates last_active_at for the given users.
// DB-side throttling: only writes rows whose last_active_at is NULL or older
// than 1 hour, keeping write volume low under high request rates.
func (s *PgStore) BatchTouchUsersLastActive(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.db.Exec(
		`UPDATE users SET last_active_at = NOW()
		 WHERE id = ANY($1)
		   AND (last_active_at IS NULL OR last_active_at < NOW() - INTERVAL '1 hour')`,
		ids,
	)
	if err != nil {
		return fmt.Errorf("store: batch touch users last active: %w", err)
	}
	return nil
}

// MarkAUPAccepted records the AUP acceptance timestamp for a user.
func (s *PgStore) MarkAUPAccepted(userID string) error {
	_, err := s.db.Exec(
		`UPDATE users SET aup_accepted_at = NOW() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("store: mark aup accepted: %w", err)
	}
	return nil
}

// isEmail checks if a string is an email address.
func isEmail(s string) bool {
	return strings.Contains(s, "@")
}
