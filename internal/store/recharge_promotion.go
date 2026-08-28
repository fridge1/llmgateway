package store

import (
	"database/sql"
	"fmt"
	"time"
)

// RechargePromotion describes a time-bounded recharge bonus campaign.
// During the [StartsAt, EndsAt) window, recharges of at least MinRechargeAmount
// receive an extra credit equal to amount * BonusRatio.
type RechargePromotion struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	StartsAt          time.Time `json:"starts_at"`
	EndsAt            time.Time `json:"ends_at"`
	BonusRatio        float64   `json:"bonus_ratio"`
	MinRechargeAmount float64   `json:"min_recharge_amount"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func scanRechargePromotion(row interface {
	Scan(dest ...any) error
}) (*RechargePromotion, error) {
	var p RechargePromotion
	if err := row.Scan(
		&p.ID, &p.Name, &p.StartsAt, &p.EndsAt,
		&p.BonusRatio, &p.MinRechargeAmount, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &p, nil
}

// CreateRechargePromotion inserts a new campaign and returns the persisted row.
func (s *PgStore) CreateRechargePromotion(p *RechargePromotion) (*RechargePromotion, error) {
	row := s.db.QueryRow(
		`INSERT INTO recharge_promotions (name, starts_at, ends_at, bonus_ratio, min_recharge_amount, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, starts_at, ends_at, bonus_ratio, min_recharge_amount, is_active, created_at, updated_at`,
		p.Name, p.StartsAt, p.EndsAt, p.BonusRatio, p.MinRechargeAmount, p.IsActive,
	)
	out, err := scanRechargePromotion(row)
	if err != nil {
		return nil, fmt.Errorf("store: create recharge promotion: %w", err)
	}
	return out, nil
}

// UpdateRechargePromotion updates an existing campaign by ID and returns the new row.
func (s *PgStore) UpdateRechargePromotion(id int, p *RechargePromotion) (*RechargePromotion, error) {
	row := s.db.QueryRow(
		`UPDATE recharge_promotions
		 SET name = $1, starts_at = $2, ends_at = $3, bonus_ratio = $4,
		     min_recharge_amount = $5, is_active = $6, updated_at = NOW()
		 WHERE id = $7
		 RETURNING id, name, starts_at, ends_at, bonus_ratio, min_recharge_amount, is_active, created_at, updated_at`,
		p.Name, p.StartsAt, p.EndsAt, p.BonusRatio, p.MinRechargeAmount, p.IsActive, id,
	)
	out, err := scanRechargePromotion(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: recharge promotion not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: update recharge promotion: %w", err)
	}
	return out, nil
}

// DeleteRechargePromotion removes a campaign by ID.
func (s *PgStore) DeleteRechargePromotion(id int) error {
	res, err := s.db.Exec(`DELETE FROM recharge_promotions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete recharge promotion: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("store: recharge promotion not found")
	}
	return nil
}

// ListRechargePromotions returns campaigns ordered by start time desc (newest first).
func (s *PgStore) ListRechargePromotions() ([]RechargePromotion, error) {
	rows, err := s.db.Query(
		`SELECT id, name, starts_at, ends_at, bonus_ratio, min_recharge_amount, is_active, created_at, updated_at
		 FROM recharge_promotions
		 ORDER BY starts_at DESC, id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list recharge promotions: %w", err)
	}
	defer rows.Close()

	var out []RechargePromotion
	for rows.Next() {
		p, err := scanRechargePromotion(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan recharge promotion: %w", err)
		}
		out = append(out, *p)
	}
	return out, nil
}

// GetBestActiveRechargePromotion picks the campaign with the highest bonus ratio
// that is active at `now` and whose threshold is satisfied by `amount`. Returns
// (nil, nil) when no campaign matches.
func (s *PgStore) GetBestActiveRechargePromotion(now time.Time, amount float64) (*RechargePromotion, error) {
	row := s.db.QueryRow(
		`SELECT id, name, starts_at, ends_at, bonus_ratio, min_recharge_amount, is_active, created_at, updated_at
		 FROM recharge_promotions
		 WHERE is_active = TRUE
		   AND starts_at <= $1
		   AND ends_at > $1
		   AND min_recharge_amount <= $2
		 ORDER BY bonus_ratio DESC, id DESC
		 LIMIT 1`,
		now, amount,
	)
	p, err := scanRechargePromotion(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get active recharge promotion: %w", err)
	}
	return p, nil
}

// GetCurrentActiveRechargePromotion picks the campaign with the highest bonus
// ratio that is active at `now`, ignoring the recharge-amount threshold so the
// UI can display the campaign even when the user hasn't picked an amount yet.
func (s *PgStore) GetCurrentActiveRechargePromotion(now time.Time) (*RechargePromotion, error) {
	row := s.db.QueryRow(
		`SELECT id, name, starts_at, ends_at, bonus_ratio, min_recharge_amount, is_active, created_at, updated_at
		 FROM recharge_promotions
		 WHERE is_active = TRUE
		   AND starts_at <= $1
		   AND ends_at > $1
		 ORDER BY bonus_ratio DESC, id DESC
		 LIMIT 1`,
		now,
	)
	p, err := scanRechargePromotion(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get current recharge promotion: %w", err)
	}
	return p, nil
}
