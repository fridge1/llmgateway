package store

import (
	"fmt"
	"time"
)

// ModerationSettings is the single-row global moderation switch.
type ModerationSettings struct {
	Enabled    bool      `json:"enabled"`
	EnforceAll bool      `json:"enforce_all"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ModerationKeyword is a banned-keyword rule.
type ModerationKeyword struct {
	ID        int       `json:"id"`
	Keyword   string    `json:"keyword"`
	Category  string    `json:"category"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// ModerationHit records a rejected request.
type ModerationHit struct {
	ID          int64     `json:"id"`
	UserID      *string   `json:"user_id"`
	TenantID    *string   `json:"tenant_id"`
	Model       string    `json:"model"`
	MatchedRule string    `json:"matched_rule"`
	Snippet     string    `json:"snippet"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *PgStore) GetModerationSettings() (*ModerationSettings, error) {
	var m ModerationSettings
	err := s.db.QueryRow(
		`SELECT enabled, enforce_all, updated_at FROM moderation_settings WHERE id=1`,
	).Scan(&m.Enabled, &m.EnforceAll, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: get moderation settings: %w", err)
	}
	return &m, nil
}

func (s *PgStore) UpdateModerationSettings(enabled, enforceAll bool) error {
	_, err := s.db.Exec(
		`UPDATE moderation_settings SET enabled=$1, enforce_all=$2, updated_at=now() WHERE id=1`,
		enabled, enforceAll)
	if err != nil {
		return fmt.Errorf("store: update moderation settings: %w", err)
	}
	return nil
}

func (s *PgStore) ListModerationKeywords() ([]ModerationKeyword, error) {
	rows, err := s.db.Query(
		`SELECT id, keyword, category, enabled, created_at FROM moderation_keywords ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("store: list moderation keywords: %w", err)
	}
	defer rows.Close()
	var list []ModerationKeyword
	for rows.Next() {
		var k ModerationKeyword
		if err := rows.Scan(&k.ID, &k.Keyword, &k.Category, &k.Enabled, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan moderation keyword: %w", err)
		}
		list = append(list, k)
	}
	return list, nil
}

func (s *PgStore) CreateModerationKeyword(keyword, category string) (*ModerationKeyword, error) {
	var k ModerationKeyword
	err := s.db.QueryRow(
		`INSERT INTO moderation_keywords (keyword, category) VALUES ($1,$2)
		 ON CONFLICT (keyword) DO UPDATE SET enabled=TRUE, category=EXCLUDED.category
		 RETURNING id, keyword, category, enabled, created_at`,
		keyword, category,
	).Scan(&k.ID, &k.Keyword, &k.Category, &k.Enabled, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create moderation keyword: %w", err)
	}
	return &k, nil
}

func (s *PgStore) DeleteModerationKeyword(id int) error {
	res, err := s.db.Exec(`DELETE FROM moderation_keywords WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("store: delete moderation keyword: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("keyword not found")
	}
	return nil
}

func (s *PgStore) CreateModerationHit(userID, tenantID *string, model, matchedRule, snippet string) error {
	_, err := s.db.Exec(
		`INSERT INTO moderation_hits (user_id, tenant_id, model, matched_rule, snippet)
		 VALUES ($1,$2,$3,$4,$5)`,
		userID, tenantID, model, matchedRule, snippet)
	if err != nil {
		return fmt.Errorf("store: create moderation hit: %w", err)
	}
	return nil
}

// ListModerationHits returns hits filtered by optional userID and time range.
func (s *PgStore) ListModerationHits(userID string, from, to *time.Time, limit, offset int) ([]ModerationHit, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	i := 1
	if userID != "" {
		where += fmt.Sprintf(" AND user_id=$%d", i)
		args = append(args, userID)
		i++
	}
	if from != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", i)
		args = append(args, *from)
		i++
	}
	if to != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", i)
		args = append(args, *to)
		i++
	}

	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM moderation_hits `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count moderation hits: %w", err)
	}
	rows, err := s.db.Query(
		`SELECT id, user_id, tenant_id, model, matched_rule, snippet, created_at
		 FROM moderation_hits `+where+
			fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, i, i+1),
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list moderation hits: %w", err)
	}
	defer rows.Close()
	var list []ModerationHit
	for rows.Next() {
		var h ModerationHit
		if err := rows.Scan(&h.ID, &h.UserID, &h.TenantID, &h.Model, &h.MatchedRule, &h.Snippet, &h.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan moderation hit: %w", err)
		}
		list = append(list, h)
	}
	return list, total, nil
}

// ListModerationEnabledTargets returns model names and tenant IDs that opted in.
func (s *PgStore) ListModerationEnabledTargets() (models []string, tenants []string, err error) {
	rows, err := s.db.Query(`SELECT name FROM models WHERE moderation_enabled = TRUE`)
	if err != nil {
		return nil, nil, fmt.Errorf("store: list moderation models: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, nil, fmt.Errorf("store: scan moderation model: %w", err)
		}
		models = append(models, name)
	}
	trows, err := s.db.Query(`SELECT id FROM tenants WHERE moderation_enabled = TRUE`)
	if err != nil {
		return nil, nil, fmt.Errorf("store: list moderation tenants: %w", err)
	}
	defer trows.Close()
	for trows.Next() {
		var id string
		if err := trows.Scan(&id); err != nil {
			return nil, nil, fmt.Errorf("store: scan moderation tenant: %w", err)
		}
		tenants = append(tenants, id)
	}
	return models, tenants, nil
}

// SetModelModeration toggles a model's moderation flag by name.
func (s *PgStore) SetModelModeration(modelName string, enabled bool) error {
	res, err := s.db.Exec(`UPDATE models SET moderation_enabled=$2 WHERE name=$1`, modelName, enabled)
	if err != nil {
		return fmt.Errorf("store: set model moderation: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("model not found")
	}
	return nil
}
