package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ReferralRule is a DB-configured referral reward rule. The latest enabled
// rule whose effective_from has passed is active; when none exists callers
// fall back to config values.
type ReferralRule struct {
	ID                  int       `json:"id"`
	InviterBonusCNY     float64   `json:"inviter_bonus_cny"`
	InviteeBonusCNY     float64   `json:"invitee_bonus_cny"`
	MinFirstRechargeCNY float64   `json:"min_first_recharge_cny"`
	Enabled             bool      `json:"enabled"`
	EffectiveFrom       time.Time `json:"effective_from"`
	CreatedBy           *string   `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
}

// ErrNoReferralRule signals that no DB rule is configured (use config fallback).
var ErrNoReferralRule = errors.New("no active referral rule")

// GetActiveReferralRule returns the newest enabled, effective rule.
func (s *PgStore) GetActiveReferralRule() (*ReferralRule, error) {
	var r ReferralRule
	err := s.db.QueryRow(
		`SELECT id, inviter_bonus_cny, invitee_bonus_cny, min_first_recharge_cny, enabled, effective_from, created_by, created_at
		 FROM referral_rules
		 WHERE enabled AND effective_from <= now()
		 ORDER BY effective_from DESC, id DESC LIMIT 1`,
	).Scan(&r.ID, &r.InviterBonusCNY, &r.InviteeBonusCNY, &r.MinFirstRechargeCNY, &r.Enabled, &r.EffectiveFrom, &r.CreatedBy, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoReferralRule
	}
	if err != nil {
		return nil, fmt.Errorf("store: get active referral rule: %w", err)
	}
	return &r, nil
}

// ListReferralRules returns rule history, newest first.
func (s *PgStore) ListReferralRules(limit int) ([]ReferralRule, error) {
	rows, err := s.db.Query(
		`SELECT id, inviter_bonus_cny, invitee_bonus_cny, min_first_recharge_cny, enabled, effective_from, created_by, created_at
		 FROM referral_rules ORDER BY effective_from DESC, id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list referral rules: %w", err)
	}
	defer rows.Close()
	var list []ReferralRule
	for rows.Next() {
		var r ReferralRule
		if err := rows.Scan(&r.ID, &r.InviterBonusCNY, &r.InviteeBonusCNY, &r.MinFirstRechargeCNY, &r.Enabled, &r.EffectiveFrom, &r.CreatedBy, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan referral rule: %w", err)
		}
		list = append(list, r)
	}
	return list, nil
}

// CreateReferralRule appends a new rule (history is append-only; the newest
// effective rule wins, so "editing" means appending a corrected rule).
func (s *PgStore) CreateReferralRule(inviterBonus, inviteeBonus, minFirstRecharge float64, enabled bool, effectiveFrom time.Time, createdBy string) (*ReferralRule, error) {
	var r ReferralRule
	var cb any
	if createdBy != "" {
		cb = createdBy
	}
	err := s.db.QueryRow(
		`INSERT INTO referral_rules (inviter_bonus_cny, invitee_bonus_cny, min_first_recharge_cny, enabled, effective_from, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id, inviter_bonus_cny, invitee_bonus_cny, min_first_recharge_cny, enabled, effective_from, created_by, created_at`,
		inviterBonus, inviteeBonus, minFirstRecharge, enabled, effectiveFrom, cb,
	).Scan(&r.ID, &r.InviterBonusCNY, &r.InviteeBonusCNY, &r.MinFirstRechargeCNY, &r.Enabled, &r.EffectiveFrom, &r.CreatedBy, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create referral rule: %w", err)
	}
	return &r, nil
}
