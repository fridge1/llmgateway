package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrAlreadyCheckedIn is returned when a user has already checked in today.
var ErrAlreadyCheckedIn = errors.New("already checked in today")

// CheckinResult describes the outcome of a successful daily check-in.
type CheckinResult struct {
	CheckinDate  time.Time `json:"checkin_date"`
	Streak       int       `json:"streak"`
	RewardCNY    float64   `json:"reward_cny"`
	BalanceAfter float64   `json:"balance_after"`
}

// CheckinStatus reports a user's current check-in state for the UI.
type CheckinStatus struct {
	CheckedInToday bool    `json:"checked_in_today"`
	CurrentStreak  int     `json:"current_streak"`
	NextRewardCNY  float64 `json:"next_reward_cny"`
}

// rewardForStreak computes the check-in reward for a given streak day using a
// 7-day escalating ladder that resets after day 7. base is the day-1 reward.
func rewardForStreak(streak int, base float64) float64 {
	if base <= 0 {
		return 0
	}
	// Ladder multipliers for days 1..7, then cycles.
	ladder := []float64{1, 2, 3, 5, 7, 10, 20}
	idx := (streak - 1) % len(ladder)
	if idx < 0 {
		idx = 0
	}
	return base * ladder[idx]
}

// DoCheckin performs an atomic daily check-in: it computes the consecutive
// streak (continuing yesterday's, otherwise resetting to 1), records the
// check-in, and credits the reward to the user's balance. Concurrency-safe via
// the (user_id, checkin_date) unique constraint.
func (s *PgStore) DoCheckin(userID string, base float64) (*CheckinResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin checkin tx: %w", err)
	}
	defer tx.Rollback()

	// Determine yesterday's streak (if checked in yesterday) to continue it.
	var prevStreak int
	var prevDate time.Time
	err = tx.QueryRow(
		`SELECT streak, checkin_date FROM daily_checkins
		 WHERE user_id = $1 ORDER BY checkin_date DESC LIMIT 1`,
		userID,
	).Scan(&prevStreak, &prevDate)
	streak := 1
	if err == nil {
		// If the most recent check-in was yesterday, continue the streak.
		if daysBetweenUTC(prevDate, time.Now().UTC()) == 1 {
			streak = prevStreak + 1
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("store: checkin prev lookup: %w", err)
	}

	reward := rewardForStreak(streak, base)

	// Insert today's check-in; unique constraint rejects double check-in.
	_, err = tx.Exec(
		`INSERT INTO daily_checkins (user_id, checkin_date, streak, reward_cny)
		 VALUES ($1, CURRENT_DATE, $2, $3)`,
		userID, streak, reward,
	)
	if err != nil {
		// Unique violation -> already checked in today.
		return nil, ErrAlreadyCheckedIn
	}

	balanceAfter := 0.0
	if reward > 0 {
		if _, err = tx.Exec(
			`UPDATE balances SET balance = balance + $1, updated_at = NOW() WHERE user_id = $2`,
			reward, userID,
		); err != nil {
			return nil, fmt.Errorf("store: checkin credit: %w", err)
		}
		if _, err = tx.Exec(
			`INSERT INTO transactions (user_id, type, amount, balance_after, description)
			 SELECT $1, 'checkin', $2, balance, $3 FROM balances WHERE user_id = $1`,
			userID, reward, fmt.Sprintf("每日签到奖励（连续 %d 天）", streak),
		); err != nil {
			return nil, fmt.Errorf("store: checkin transaction: %w", err)
		}
	}
	if err = tx.QueryRow(
		`SELECT balance FROM balances WHERE user_id = $1`, userID,
	).Scan(&balanceAfter); err != nil {
		return nil, fmt.Errorf("store: checkin get balance: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit checkin: %w", err)
	}
	return &CheckinResult{
		CheckinDate:  time.Now().UTC(),
		Streak:       streak,
		RewardCNY:    reward,
		BalanceAfter: balanceAfter,
	}, nil
}

// GetCheckinStatus returns whether the user checked in today, their current
// streak, and the reward they would earn on their next check-in.
func (s *PgStore) GetCheckinStatus(userID string, base float64) (*CheckinStatus, error) {
	var prevStreak int
	var prevDate time.Time
	err := s.db.QueryRow(
		`SELECT streak, checkin_date FROM daily_checkins
		 WHERE user_id = $1 ORDER BY checkin_date DESC LIMIT 1`,
		userID,
	).Scan(&prevStreak, &prevDate)
	if errors.Is(err, sql.ErrNoRows) {
		return &CheckinStatus{CheckedInToday: false, CurrentStreak: 0, NextRewardCNY: rewardForStreak(1, base)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: checkin status: %w", err)
	}

	gap := daysBetweenUTC(prevDate, time.Now().UTC())
	switch {
	case gap == 0:
		// Already checked in today.
		return &CheckinStatus{CheckedInToday: true, CurrentStreak: prevStreak, NextRewardCNY: rewardForStreak(prevStreak+1, base)}, nil
	case gap == 1:
		// Streak continues on next check-in.
		return &CheckinStatus{CheckedInToday: false, CurrentStreak: prevStreak, NextRewardCNY: rewardForStreak(prevStreak+1, base)}, nil
	default:
		// Streak broken; next check-in resets to day 1.
		return &CheckinStatus{CheckedInToday: false, CurrentStreak: 0, NextRewardCNY: rewardForStreak(1, base)}, nil
	}
}

// daysBetweenUTC returns the whole-day difference between two dates (date-only,
// UTC). Both inputs are truncated to midnight before comparison.
func daysBetweenUTC(earlier, later time.Time) int {
	e := time.Date(earlier.Year(), earlier.Month(), earlier.Day(), 0, 0, 0, 0, time.UTC)
	l := time.Date(later.Year(), later.Month(), later.Day(), 0, 0, 0, 0, time.UTC)
	return int(l.Sub(e).Hours() / 24)
}
