package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// ReferralInfo summarizes a user's invite code and referral performance.
type ReferralInfo struct {
	ReferralCode  string  `json:"referral_code"`
	InvitedCount  int     `json:"invited_count"`
	RewardedCount int     `json:"rewarded_count"`
	TotalReward   float64 `json:"total_reward"`
}

// GetReferralCode returns the user's invite code, generating/persisting one if
// the column is somehow NULL (defensive; the migration backfills all rows).
func (s *PgStore) GetReferralCode(userID string) (string, error) {
	var code sql.NullString
	if err := s.db.QueryRow(
		`SELECT referral_code FROM users WHERE id = $1`, userID,
	).Scan(&code); err != nil {
		return "", fmt.Errorf("store: get referral code: %w", err)
	}
	if code.Valid && code.String != "" {
		return code.String, nil
	}
	var generated string
	if err := s.db.QueryRow(
		`UPDATE users SET referral_code = UPPER(SUBSTRING(MD5(id::text) FROM 1 FOR 8))
		 WHERE id = $1 RETURNING referral_code`, userID,
	).Scan(&generated); err != nil {
		return "", fmt.Errorf("store: backfill referral code: %w", err)
	}
	return generated, nil
}

// GetUserIDByReferralCode resolves an invite code to a user ID. Returns
// sql.ErrNoRows if the code doesn't exist.
func (s *PgStore) GetUserIDByReferralCode(code string) (string, error) {
	var id string
	err := s.db.QueryRow(
		`SELECT id FROM users WHERE referral_code = $1`, code,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// SetReferredBy records who invited a newly registered user. No-op if the user
// already has a referrer or if referrerID equals userID (self-invite guard).
func (s *PgStore) SetReferredBy(userID, referrerID string) error {
	if userID == referrerID {
		return nil
	}
	_, err := s.db.Exec(
		`UPDATE users SET referred_by = $2
		 WHERE id = $1 AND referred_by IS NULL`,
		userID, referrerID,
	)
	if err != nil {
		return fmt.Errorf("store: set referred_by: %w", err)
	}
	return nil
}

// GrantReferralReward credits both the inviter and the invited user when the
// invited user completes their first recharge. Idempotent via the
// referral_reward_granted flag on the invited user. Returns true if a reward
// was granted this call (false if no referrer, already granted, or amounts 0).
func (s *PgStore) GrantReferralReward(invitedUserID string, inviterBonus, inviteeBonus float64) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("store: begin referral reward tx: %w", err)
	}
	defer tx.Rollback()

	// Lock the invited user's row; verify referrer exists and reward not granted.
	var referrer sql.NullString
	var granted bool
	err = tx.QueryRow(
		`SELECT referred_by, referral_reward_granted FROM users WHERE id = $1 FOR UPDATE`,
		invitedUserID,
	).Scan(&referrer, &granted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: referral reward lookup: %w", err)
	}
	if granted || !referrer.Valid || referrer.String == "" {
		return false, nil
	}

	// Mark granted first so concurrent calls short-circuit.
	if _, err = tx.Exec(
		`UPDATE users SET referral_reward_granted = TRUE WHERE id = $1`, invitedUserID,
	); err != nil {
		return false, fmt.Errorf("store: mark referral granted: %w", err)
	}

	if err = creditWithinTx(tx, referrer.String, inviterBonus, "邀请好友充值奖励"); err != nil {
		return false, err
	}
	if err = creditWithinTx(tx, invitedUserID, inviteeBonus, "受邀注册首充奖励"); err != nil {
		return false, err
	}

	if err = tx.Commit(); err != nil {
		return false, fmt.Errorf("store: commit referral reward: %w", err)
	}
	return true, nil
}

// creditWithinTx adds amount to a user's balance and records a transaction,
// reusing the caller's transaction. No-op when amount <= 0.
func creditWithinTx(tx *sql.Tx, userID string, amount float64, desc string) error {
	if amount <= 0 {
		return nil
	}
	if _, err := tx.Exec(
		`UPDATE balances SET balance = balance + $1, updated_at = NOW() WHERE user_id = $2`,
		amount, userID,
	); err != nil {
		return fmt.Errorf("store: referral credit: %w", err)
	}
	if _, err := tx.Exec(
		`INSERT INTO transactions (user_id, type, amount, balance_after, description)
		 SELECT $1, 'referral_bonus', $2, balance, $3 FROM balances WHERE user_id = $1`,
		userID, amount, desc,
	); err != nil {
		return fmt.Errorf("store: referral transaction: %w", err)
	}
	return nil
}

// GetReferralInfo returns the user's invite code plus invitation stats.
func (s *PgStore) GetReferralInfo(userID string) (*ReferralInfo, error) {
	code, err := s.GetReferralCode(userID)
	if err != nil {
		return nil, err
	}
	info := &ReferralInfo{ReferralCode: code}

	if err := s.db.QueryRow(
		`SELECT
		   COUNT(*),
		   COUNT(*) FILTER (WHERE referral_reward_granted)
		 FROM users WHERE referred_by = $1`,
		userID,
	).Scan(&info.InvitedCount, &info.RewardedCount); err != nil {
		return nil, fmt.Errorf("store: referral stats: %w", err)
	}

	if err := s.db.QueryRow(
		`SELECT COALESCE(SUM(amount), 0) FROM transactions
		 WHERE user_id = $1 AND type = 'referral_bonus'`,
		userID,
	).Scan(&info.TotalReward); err != nil {
		return nil, fmt.Errorf("store: referral total reward: %w", err)
	}
	return info, nil
}
