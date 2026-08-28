package store

import (
	"database/sql"
	"fmt"
	"math/rand/v2"
	"time"
)

// RechargeLottery holds the configuration for the recharge-count lottery.
type RechargeLottery struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	TriggerEvery int       `json:"trigger_every"`
	TotalRounds  int       `json:"total_rounds"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RechargeLotteryRound holds the result of a single draw round.
type RechargeLotteryRound struct {
	ID               int64     `json:"id"`
	LotteryID        int       `json:"lottery_id"`
	RoundNo          int       `json:"round_no"`
	WinnerUserID     string    `json:"winner_user_id"`
	WinnerNickname   string    `json:"winner_nickname,omitempty"`
	WinnerPhone      string    `json:"winner_phone,omitempty"`
	WinnerAmount     float64   `json:"winner_amount"`
	WinnerOrderNo    string    `json:"winner_order_no"`
	ParticipantCount int       `json:"participant_count"`
	CreatedAt        time.Time `json:"created_at"`
}

// RechargeLotteryWin is returned when a draw is triggered and a winner is picked.
type RechargeLotteryWin struct {
	RoundNo      int
	WinnerUserID string
	WinnerAmount float64
	OrderNo      string
}

// GetActiveLottery returns the single active recharge lottery, or nil if none.
func (s *PgStore) GetActiveLottery() (*RechargeLottery, error) {
	var l RechargeLottery
	err := s.db.QueryRow(
		`SELECT id, name, status, trigger_every, total_rounds, created_at, updated_at
		 FROM recharge_lottery WHERE status = 'active' LIMIT 1`,
	).Scan(&l.ID, &l.Name, &l.Status, &l.TriggerEvery, &l.TotalRounds, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get active lottery: %w", err)
	}
	return &l, nil
}

// RecordEntryAndMaybeDraw records this recharge order as a lottery entry and
// triggers a draw if the pending count reaches trigger_every.
// Returns non-nil RechargeLotteryWin when a draw was completed. Idempotent via order_no UNIQUE.
func (s *PgStore) RecordEntryAndMaybeDraw(lotteryID int, userID, orderNo string, amount float64) (*RechargeLotteryWin, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: recharge lottery: begin: %w", err)
	}
	defer tx.Rollback()

	// Lock the lottery row to serialize concurrent entries.
	var triggerEvery, totalRounds int
	err = tx.QueryRow(
		`SELECT trigger_every, total_rounds FROM recharge_lottery WHERE id = $1 AND status = 'active' FOR UPDATE`,
		lotteryID,
	).Scan(&triggerEvery, &totalRounds)
	if err == sql.ErrNoRows {
		return nil, nil // lottery paused or deleted between check and here
	}
	if err != nil {
		return nil, fmt.Errorf("store: recharge lottery: lock: %w", err)
	}

	// Insert entry; ON CONFLICT silently skips duplicate order_no (idempotent).
	_, err = tx.Exec(
		`INSERT INTO recharge_lottery_entries (lottery_id, user_id, order_no, amount)
		 VALUES ($1, $2, $3, $4) ON CONFLICT (order_no) DO NOTHING`,
		lotteryID, userID, orderNo, amount,
	)
	if err != nil {
		return nil, fmt.Errorf("store: recharge lottery: insert entry: %w", err)
	}

	// Count pending entries (those not yet assigned to a round).
	var pendingCount int
	err = tx.QueryRow(
		`SELECT COUNT(*) FROM recharge_lottery_entries WHERE lottery_id = $1 AND round_no IS NULL`,
		lotteryID,
	).Scan(&pendingCount)
	if err != nil {
		return nil, fmt.Errorf("store: recharge lottery: count pending: %w", err)
	}

	if pendingCount < triggerEvery {
		return nil, tx.Commit()
	}

	// Fetch the oldest N pending entries for this round.
	rows, err := tx.Query(
		`SELECT id, user_id, order_no, amount FROM recharge_lottery_entries
		 WHERE lottery_id = $1 AND round_no IS NULL
		 ORDER BY id LIMIT $2 FOR UPDATE`,
		lotteryID, triggerEvery,
	)
	if err != nil {
		return nil, fmt.Errorf("store: recharge lottery: fetch entries: %w", err)
	}
	type entry struct {
		id      int64
		userID  string
		orderNo string
		amount  float64
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.userID, &e.orderNo, &e.amount); err != nil {
			rows.Close()
			return nil, fmt.Errorf("store: recharge lottery: scan entry: %w", err)
		}
		entries = append(entries, e)
	}
	rows.Close()
	if len(entries) < triggerEvery {
		// Race: another goroutine won the lock and already consumed them.
		return nil, tx.Commit()
	}

	// Pick a random winner from the lowest 5 amounts.
	// Sort entries by amount ascending to get the lowest amounts first.
	type indexedEntry struct {
		entry
		originalIndex int
	}
	indexed := make([]indexedEntry, len(entries))
	for i, e := range entries {
		indexed[i] = indexedEntry{entry: e, originalIndex: i}
	}
	// Simple bubble sort by amount (ascending)
	for i := 0; i < len(indexed)-1; i++ {
		for j := 0; j < len(indexed)-i-1; j++ {
			if indexed[j].amount > indexed[j+1].amount {
				indexed[j], indexed[j+1] = indexed[j+1], indexed[j]
			}
		}
	}
	// Take the lowest 5 (or fewer if less than 5 entries)
	poolSize := 5
	if len(indexed) < poolSize {
		poolSize = len(indexed)
	}
	lowestPool := indexed[:poolSize]
	// Pick a random winner from the lowest pool
	winner := lowestPool[rand.IntN(len(lowestPool))].entry
	nextRound := totalRounds + 1

	// Mark all N entries as belonging to this round.
	ids := make([]int64, len(entries))
	for i, e := range entries {
		ids[i] = e.id
	}
	_, err = tx.Exec(
		`UPDATE recharge_lottery_entries SET round_no = $1 WHERE lottery_id = $2 AND id = ANY($3)`,
		nextRound, lotteryID, ids,
	)
	if err != nil {
		return nil, fmt.Errorf("store: recharge lottery: mark round: %w", err)
	}

	// Record the round result.
	_, err = tx.Exec(
		`INSERT INTO recharge_lottery_rounds
		 (lottery_id, round_no, winner_user_id, winner_amount, winner_order_no, participant_count)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		lotteryID, nextRound, winner.userID, winner.amount, winner.orderNo, len(entries),
	)
	if err != nil {
		return nil, fmt.Errorf("store: recharge lottery: insert round: %w", err)
	}

	// Advance the total_rounds counter.
	_, err = tx.Exec(
		`UPDATE recharge_lottery SET total_rounds = $1, updated_at = NOW() WHERE id = $2`,
		nextRound, lotteryID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: recharge lottery: update rounds: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: recharge lottery: commit: %w", err)
	}
	return &RechargeLotteryWin{
		RoundNo:      nextRound,
		WinnerUserID: winner.userID,
		WinnerAmount: winner.amount,
		OrderNo:      winner.orderNo,
	}, nil
}

// GetCurrentRoundEntryCount returns the number of pending (not-yet-drawn) entries for a lottery.
func (s *PgStore) GetCurrentRoundEntryCount(lotteryID int) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM recharge_lottery_entries WHERE lottery_id = $1 AND round_no IS NULL`,
		lotteryID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: get current round entry count: %w", err)
	}
	return count, nil
}

// CreateRechargeLottery creates a new lottery configuration.
func (s *PgStore) CreateRechargeLottery(name string, triggerEvery int) (*RechargeLottery, error) {
	var l RechargeLottery
	err := s.db.QueryRow(
		`INSERT INTO recharge_lottery (name, trigger_every)
		 VALUES ($1, $2)
		 RETURNING id, name, status, trigger_every, total_rounds, created_at, updated_at`,
		name, triggerEvery,
	).Scan(&l.ID, &l.Name, &l.Status, &l.TriggerEvery, &l.TotalRounds, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create recharge lottery: %w", err)
	}
	return &l, nil
}

// UpdateRechargeLottery updates an existing lottery configuration.
func (s *PgStore) UpdateRechargeLottery(id int, name, status string, triggerEvery int) (*RechargeLottery, error) {
	var l RechargeLottery
	err := s.db.QueryRow(
		`UPDATE recharge_lottery
		 SET name = $1, status = $2, trigger_every = $3, updated_at = NOW()
		 WHERE id = $4
		 RETURNING id, name, status, trigger_every, total_rounds, created_at, updated_at`,
		name, status, triggerEvery, id,
	).Scan(&l.ID, &l.Name, &l.Status, &l.TriggerEvery, &l.TotalRounds, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: recharge lottery not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: update recharge lottery: %w", err)
	}
	return &l, nil
}

// ListRechargeLotteryRounds returns paginated draw history for a lottery.
func (s *PgStore) ListRechargeLotteryRounds(lotteryID, limit, offset int) ([]RechargeLotteryRound, int, error) {
	var total int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM recharge_lottery_rounds WHERE lottery_id = $1`, lotteryID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: list rounds count: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT r.id, r.lottery_id, r.round_no, r.winner_user_id,
		        COALESCE(u.nickname, ''), r.winner_amount, r.winner_order_no, r.participant_count, r.created_at
		 FROM recharge_lottery_rounds r
		 LEFT JOIN users u ON u.id = r.winner_user_id
		 WHERE r.lottery_id = $1
		 ORDER BY r.round_no DESC
		 LIMIT $2 OFFSET $3`,
		lotteryID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list rounds: %w", err)
	}
	defer rows.Close()

	var rounds []RechargeLotteryRound
	for rows.Next() {
		var r RechargeLotteryRound
		if err := rows.Scan(&r.ID, &r.LotteryID, &r.RoundNo, &r.WinnerUserID, &r.WinnerNickname,
			&r.WinnerAmount, &r.WinnerOrderNo, &r.ParticipantCount, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan round: %w", err)
		}
		rounds = append(rounds, r)
	}
	if rounds == nil {
		rounds = []RechargeLotteryRound{}
	}
	return rounds, total, nil
}

// ListRechargeLotteryRoundsAdmin is the admin-only variant of ListRechargeLotteryRounds
// that additionally returns the winner's phone number (PII). It MUST only be called from
// authenticated admin endpoints — the public-facing ListRechargeLotteryRounds intentionally
// omits the phone number to avoid exposing PII to unauthenticated visitors.
func (s *PgStore) ListRechargeLotteryRoundsAdmin(lotteryID, limit, offset int) ([]RechargeLotteryRound, int, error) {
	var total int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM recharge_lottery_rounds WHERE lottery_id = $1`, lotteryID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: list rounds count: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT r.id, r.lottery_id, r.round_no, r.winner_user_id,
		        COALESCE(u.nickname, ''), COALESCE(u.phone, ''), r.winner_amount, r.winner_order_no, r.participant_count, r.created_at
		 FROM recharge_lottery_rounds r
		 LEFT JOIN users u ON u.id = r.winner_user_id
		 WHERE r.lottery_id = $1
		 ORDER BY r.round_no DESC
		 LIMIT $2 OFFSET $3`,
		lotteryID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list rounds admin: %w", err)
	}
	defer rows.Close()

	var rounds []RechargeLotteryRound
	for rows.Next() {
		var r RechargeLotteryRound
		if err := rows.Scan(&r.ID, &r.LotteryID, &r.RoundNo, &r.WinnerUserID, &r.WinnerNickname, &r.WinnerPhone,
			&r.WinnerAmount, &r.WinnerOrderNo, &r.ParticipantCount, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan round admin: %w", err)
		}
		rounds = append(rounds, r)
	}
	if rounds == nil {
		rounds = []RechargeLotteryRound{}
	}
	return rounds, total, nil
}
