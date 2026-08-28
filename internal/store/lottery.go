package store

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

// LotteryEvent represents a lottery campaign.
type LotteryEvent struct {
	ID                 int        `json:"id"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	Status             string     `json:"status"`
	MinRechargeCNY     float64    `json:"min_recharge_cny"`
	MinOrderCountToDraw int       `json:"min_order_count_to_draw"`
	StartTime          *time.Time `json:"start_time"`
	EndTime            *time.Time `json:"end_time"`
	ParticipantCount   int        `json:"participant_count"`
	DrawnAt            *time.Time `json:"drawn_at,omitempty"`
	DrawnBy            *string    `json:"drawn_by,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// LotteryPrize represents a prize in a lottery event.
type LotteryPrize struct {
	ID             int       `json:"id"`
	EventID        int       `json:"event_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Weight         int       `json:"weight"`
	TotalStock     int       `json:"total_stock"`
	RemainingStock int       `json:"remaining_stock"`
	PrizeType      string    `json:"prize_type"`
	PrizeValue     float64   `json:"prize_value"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// LotteryRecord represents a single draw result.
type LotteryRecord struct {
	ID             int64     `json:"id"`
	EventID        int       `json:"event_id"`
	UserID         string    `json:"user_id"`
	Phone          string    `json:"phone"`
	Nickname       string    `json:"nickname"`
	PrizeID        *int      `json:"prize_id"`
	PrizeName      string    `json:"prize_name"`
	PrizeType      string    `json:"prize_type"`
	PrizeValue     float64   `json:"prize_value"`
	OrderNo        string    `json:"order_no"`
	RechargeAmount float64   `json:"recharge_amount"`
	CreatedAt      time.Time `json:"created_at"`
}

// PublicLotteryRecord represents a draw result safe for public display.
type PublicLotteryRecord struct {
	ID          int64     `json:"id"`
	MaskedPhone string    `json:"masked_phone"`
	PrizeName   string    `json:"prize_name"`
	PrizeType   string    `json:"prize_type"`
	PrizeValue  float64   `json:"prize_value"`
	CreatedAt   time.Time `json:"created_at"`
}

func maskLotteryPhone(phone string) string {
	if len(phone) != 11 {
		return "****"
	}
	for i := 0; i < len(phone); i++ {
		if phone[i] < '0' || phone[i] > '9' {
			return "****"
		}
	}
	return phone[:3] + "****" + phone[7:]
}

// CreateLotteryEvent creates a new lottery event.
func (s *PgStore) CreateLotteryEvent(name, description string, status string, minRechargeCNY float64, minOrderCountToDraw int, startTime, endTime *time.Time) (*LotteryEvent, error) {
	var e LotteryEvent
	err := s.db.QueryRow(
		`INSERT INTO lottery_events (name, description, status, min_recharge_cny, min_order_count_to_draw, start_time, end_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, name, description, status, min_recharge_cny, min_order_count_to_draw, start_time, end_time, created_at, updated_at`,
		name, description, status, minRechargeCNY, minOrderCountToDraw, startTime, endTime,
	).Scan(&e.ID, &e.Name, &e.Description, &e.Status, &e.MinRechargeCNY, &e.MinOrderCountToDraw, &e.StartTime, &e.EndTime, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create lottery event: %w", err)
	}
	return &e, nil
}

// UpdateLotteryEvent updates an existing lottery event.
//
// status must be one of "active", "paused", "ended". A finalized event (one
// that has already been drawn, or is "ended") cannot be re-activated: this
// prevents re-opening a campaign after winners have been picked and balances
// credited.
func (s *PgStore) UpdateLotteryEvent(id int, name, description, status string, minRechargeCNY float64, minOrderCountToDraw int, startTime, endTime *time.Time) (*LotteryEvent, error) {
	switch status {
	case "active", "paused", "ended":
	default:
		return nil, fmt.Errorf("store: invalid lottery status %q", status)
	}

	// Reject re-activating an already-drawn or ended event.
	var prevStatus string
	var prevDrawnAt *time.Time
	if err := s.db.QueryRow(
		`SELECT status, drawn_at FROM lottery_events WHERE id = $1`,
		id,
	).Scan(&prevStatus, &prevDrawnAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("store: lottery event not found")
		}
		return nil, fmt.Errorf("store: load lottery event for update: %w", err)
	}
	if status == "active" && (prevStatus == "ended" || prevDrawnAt != nil) {
		return nil, fmt.Errorf("store: cannot re-activate a drawn or ended event")
	}

	var e LotteryEvent
	err := s.db.QueryRow(
		`UPDATE lottery_events
		 SET name = $1, description = $2, status = $3, min_recharge_cny = $4, min_order_count_to_draw = $5, start_time = $6, end_time = $7, updated_at = NOW()
		 WHERE id = $8
		 RETURNING id, name, description, status, min_recharge_cny, min_order_count_to_draw, start_time, end_time, created_at, updated_at`,
		name, description, status, minRechargeCNY, minOrderCountToDraw, startTime, endTime, id,
	).Scan(&e.ID, &e.Name, &e.Description, &e.Status, &e.MinRechargeCNY, &e.MinOrderCountToDraw, &e.StartTime, &e.EndTime, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: lottery event not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: update lottery event: %w", err)
	}
	return &e, nil
}

// ListLotteryEvents lists all lottery events with pagination.
func (s *PgStore) ListLotteryEvents(limit, offset int) ([]LotteryEvent, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM lottery_events`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count lottery events: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT e.id, e.name, e.description, e.status, e.min_recharge_cny, e.min_order_count_to_draw, e.start_time, e.end_time,
		        COALESCE(COUNT(r.id), 0) as participant_count, e.drawn_at, e.drawn_by, e.created_at, e.updated_at
		 FROM lottery_events e
		 LEFT JOIN lottery_records r ON e.id = r.event_id
		 GROUP BY e.id
		 ORDER BY e.id DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list lottery events: %w", err)
	}
	defer rows.Close()

	var events []LotteryEvent
	for rows.Next() {
		var e LotteryEvent
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.Status, &e.MinRechargeCNY, &e.MinOrderCountToDraw, &e.StartTime, &e.EndTime, &e.ParticipantCount, &e.DrawnAt, &e.DrawnBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan lottery event: %w", err)
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

// GetLotteryEvent gets a single lottery event by ID.
func (s *PgStore) GetLotteryEvent(id int) (*LotteryEvent, error) {
	var e LotteryEvent
	err := s.db.QueryRow(
		`SELECT id, name, description, status, min_recharge_cny, min_order_count_to_draw, start_time, end_time, drawn_at, drawn_by, created_at, updated_at
		 FROM lottery_events WHERE id = $1`,
		id,
	).Scan(&e.ID, &e.Name, &e.Description, &e.Status, &e.MinRechargeCNY, &e.MinOrderCountToDraw, &e.StartTime, &e.EndTime, &e.DrawnAt, &e.DrawnBy, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get lottery event: %w", err)
	}
	return &e, nil
}

// CreateLotteryPrize creates a new prize for an event.
func (s *PgStore) CreateLotteryPrize(eventID int, name, description string, weight, totalStock int, prizeType string, prizeValue float64, sortOrder int) (*LotteryPrize, error) {
	var p LotteryPrize
	err := s.db.QueryRow(
		`INSERT INTO lottery_prizes (event_id, name, description, weight, total_stock, remaining_stock, prize_type, prize_value, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8)
		 RETURNING id, event_id, name, description, weight, total_stock, remaining_stock, prize_type, prize_value, sort_order, created_at, updated_at`,
		eventID, name, description, weight, totalStock, prizeType, prizeValue, sortOrder,
	).Scan(&p.ID, &p.EventID, &p.Name, &p.Description, &p.Weight, &p.TotalStock, &p.RemainingStock, &p.PrizeType, &p.PrizeValue, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create lottery prize: %w", err)
	}
	return &p, nil
}

// UpdateLotteryPrize updates an existing prize.
func (s *PgStore) UpdateLotteryPrize(id int, name, description string, weight, totalStock int, prizeType string, prizeValue float64, sortOrder int) (*LotteryPrize, error) {
	var p LotteryPrize
	err := s.db.QueryRow(
		`UPDATE lottery_prizes
		 SET name = $1, description = $2, weight = $3, total_stock = $4, remaining_stock = $4, prize_type = $5, prize_value = $6, sort_order = $7, updated_at = NOW()
		 WHERE id = $8
		 RETURNING id, event_id, name, description, weight, total_stock, remaining_stock, prize_type, prize_value, sort_order, created_at, updated_at`,
		name, description, weight, totalStock, prizeType, prizeValue, sortOrder, id,
	).Scan(&p.ID, &p.EventID, &p.Name, &p.Description, &p.Weight, &p.TotalStock, &p.RemainingStock, &p.PrizeType, &p.PrizeValue, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: lottery prize not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: update lottery prize: %w", err)
	}
	return &p, nil
}

// DeleteLotteryPrize deletes a prize.
func (s *PgStore) DeleteLotteryPrize(id int) error {
	res, err := s.db.Exec(`DELETE FROM lottery_prizes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete lottery prize: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: lottery prize not found")
	}
	return nil
}

// ListLotteryPrizes lists all prizes for an event, ordered by sort_order.
func (s *PgStore) ListLotteryPrizes(eventID int) ([]LotteryPrize, error) {
	rows, err := s.db.Query(
		`SELECT id, event_id, name, description, weight, total_stock, remaining_stock, prize_type, prize_value, sort_order, created_at, updated_at
		 FROM lottery_prizes
		 WHERE event_id = $1
		 ORDER BY sort_order, id`,
		eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list lottery prizes: %w", err)
	}
	defer rows.Close()

	var prizes []LotteryPrize
	for rows.Next() {
		var p LotteryPrize
		if err := rows.Scan(&p.ID, &p.EventID, &p.Name, &p.Description, &p.Weight, &p.TotalStock, &p.RemainingStock, &p.PrizeType, &p.PrizeValue, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan lottery prize: %w", err)
		}
		prizes = append(prizes, p)
	}
	return prizes, rows.Err()
}

// ListLotteryRecords lists draw records for an event with pagination, joined with prize info.
func (s *PgStore) ListLotteryRecords(eventID, limit, offset int) ([]LotteryRecord, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM lottery_records WHERE event_id = $1`, eventID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count lottery records: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT r.id, r.event_id, r.user_id, COALESCE(u.phone, ''), COALESCE(u.nickname, ''), r.prize_id, COALESCE(p.name, ''), COALESCE(p.prize_type, ''), COALESCE(p.prize_value, 0), r.order_no, r.recharge_amount, r.created_at
		 FROM lottery_records r
		 LEFT JOIN lottery_prizes p ON p.id = r.prize_id
		 LEFT JOIN users u ON u.id::text = r.user_id
		 WHERE r.event_id = $1
		 ORDER BY r.id DESC
		 LIMIT $2 OFFSET $3`,
		eventID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list lottery records: %w", err)
	}
	defer rows.Close()

	var records []LotteryRecord
	for rows.Next() {
		var r LotteryRecord
		if err := rows.Scan(&r.ID, &r.EventID, &r.UserID, &r.Phone, &r.Nickname, &r.PrizeID, &r.PrizeName, &r.PrizeType, &r.PrizeValue, &r.OrderNo, &r.RechargeAmount, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan lottery record: %w", err)
		}
		records = append(records, r)
	}
	return records, total, rows.Err()
}

// ListUserLotteryRecords lists a user's own participation records across all
// events, joined with the resolved prize info and event name. Returns
// user-scoped data safe for the current user to view (no other users' rows).
func (s *PgStore) ListUserLotteryRecords(userID string, limit, offset int) ([]LotteryRecord, int, error) {
	var total int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM lottery_records WHERE user_id = $1`,
		userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count user lottery records: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT r.id, r.event_id, r.user_id, COALESCE(u.phone, ''), COALESCE(u.nickname, ''),
		        r.prize_id, COALESCE(p.name, ''), COALESCE(p.prize_type, ''), COALESCE(p.prize_value, 0),
		        r.order_no, r.recharge_amount, r.created_at
		 FROM lottery_records r
		 LEFT JOIN lottery_prizes p ON p.id = r.prize_id
		 LEFT JOIN users u ON u.id::text = r.user_id
		 WHERE r.user_id = $1
		 ORDER BY r.created_at DESC, r.id DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list user lottery records: %w", err)
	}
	defer rows.Close()

	records := make([]LotteryRecord, 0)
	for rows.Next() {
		var r LotteryRecord
		if err := rows.Scan(&r.ID, &r.EventID, &r.UserID, &r.Phone, &r.Nickname, &r.PrizeID, &r.PrizeName, &r.PrizeType, &r.PrizeValue, &r.OrderNo, &r.RechargeAmount, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan user lottery record: %w", err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: iterate user lottery records: %w", err)
	}
	return records, total, nil
}

// ListPublicLotteryWinners lists public-safe winning records for an event.
func (s *PgStore) ListPublicLotteryWinners(eventID, limit, offset int) ([]PublicLotteryRecord, int, error) {
	var total int
	if err := s.db.QueryRow(
		`SELECT COUNT(*)
		 FROM lottery_records r
		 INNER JOIN lottery_prizes p ON p.id = r.prize_id
		 INNER JOIN users u ON u.id::text = r.user_id
		 WHERE r.event_id = $1 AND p.prize_type <> 'none'`,
		eventID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count public lottery winners: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT r.id, u.phone, p.name, p.prize_type,
		        CASE WHEN p.prize_type = 'match_recharge' THEN r.recharge_amount ELSE p.prize_value END,
		        r.created_at
		 FROM lottery_records r
		 INNER JOIN lottery_prizes p ON p.id = r.prize_id
		 INNER JOIN users u ON u.id::text = r.user_id
		 WHERE r.event_id = $1 AND p.prize_type <> 'none'
		 ORDER BY r.created_at DESC, r.id DESC
		 LIMIT $2 OFFSET $3`,
		eventID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list public lottery winners: %w", err)
	}
	defer rows.Close()

	records := make([]PublicLotteryRecord, 0)
	for rows.Next() {
		var record PublicLotteryRecord
		var phone string
		if err := rows.Scan(&record.ID, &phone, &record.PrizeName, &record.PrizeType, &record.PrizeValue, &record.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan public lottery winner: %w", err)
		}
		record.MaskedPhone = maskLotteryPhone(phone)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: iterate public lottery winners: %w", err)
	}
	return records, total, nil
}

// ListAllPublicLotteryWinners lists public-safe winning records across all events,
// ordered by created_at DESC. Used for the historical winner board on /lottery.
func (s *PgStore) ListAllPublicLotteryWinners(limit, offset int) ([]PublicLotteryRecord, int, error) {
	var total int
	if err := s.db.QueryRow(
		`SELECT COUNT(*)
		 FROM lottery_records r
		 INNER JOIN lottery_prizes p ON p.id = r.prize_id
		 INNER JOIN users u ON u.id::text = r.user_id
		 WHERE p.prize_type <> 'none'`,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count public lottery winners: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT r.id, u.phone, p.name, p.prize_type,
		        CASE WHEN p.prize_type = 'match_recharge' THEN r.recharge_amount ELSE p.prize_value END,
		        r.created_at
		 FROM lottery_records r
		 INNER JOIN lottery_prizes p ON p.id = r.prize_id
		 INNER JOIN users u ON u.id::text = r.user_id
		 WHERE p.prize_type <> 'none'
		 ORDER BY r.created_at DESC, r.id DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list public lottery winners: %w", err)
	}
	defer rows.Close()

	records := make([]PublicLotteryRecord, 0)
	for rows.Next() {
		var record PublicLotteryRecord
		var phone string
		if err := rows.Scan(&record.ID, &phone, &record.PrizeName, &record.PrizeType, &record.PrizeValue, &record.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan public lottery winner: %w", err)
		}
		record.MaskedPhone = maskLotteryPhone(phone)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: iterate public lottery winners: %w", err)
	}
	return records, total, nil
}

// RecordLotteryParticipation records user participation in lottery (no immediate draw).
func (s *PgStore) RecordLotteryParticipation(userID, orderNo string, rechargeAmount float64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	// Find active event that matches recharge amount and time range.
	var eventID int
	err = tx.QueryRow(
		`SELECT id FROM lottery_events
		 WHERE status = 'active'
		   AND min_recharge_cny <= $1
		   AND (start_time IS NULL OR start_time <= $2)
		   AND (end_time IS NULL OR end_time >= $2)
		 LIMIT 1`,
		rechargeAmount, now,
	).Scan(&eventID)
	if err == sql.ErrNoRows {
		// No active event, just return (not an error).
		return nil
	}
	if err != nil {
		return fmt.Errorf("store: find active lottery event: %w", err)
	}

	// Insert participation record (prize_id = NULL, will be filled on draw).
	_, err = tx.Exec(
		`INSERT INTO lottery_records (event_id, user_id, order_no, recharge_amount)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (order_no) DO NOTHING`,
		eventID, userID, orderNo, rechargeAmount,
	)
	if err != nil {
		return fmt.Errorf("store: insert lottery participation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit lottery participation: %w", err)
	}

	return nil
}

// DrawLottery performs an automatic lottery draw on qualifying recharge (idempotent by order_no).
// DEPRECATED: Use RecordLotteryParticipation + DrawEventLottery instead.
func (s *PgStore) DrawLottery(userID, orderNo string, rechargeAmount float64) (*LotteryRecord, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	// Find active event that matches recharge amount and time range.
	var eventID int
	err = tx.QueryRow(
		`SELECT id FROM lottery_events
		 WHERE status = 'active'
		   AND min_recharge_cny <= $1
		   AND (start_time IS NULL OR start_time <= $2)
		   AND (end_time IS NULL OR end_time >= $2)
		 LIMIT 1`,
		rechargeAmount, now,
	).Scan(&eventID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: find active lottery event: %w", err)
	}

	// Try insert record (idempotent by unique index on order_no).
	var recordID int64
	err = tx.QueryRow(
		`INSERT INTO lottery_records (event_id, user_id, order_no, recharge_amount)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (order_no) DO NOTHING
		 RETURNING id`,
		eventID, userID, orderNo, rechargeAmount,
	).Scan(&recordID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: insert lottery record: %w", err)
	}

	// Load available prizes (unlimited stock or remaining > 0).
	rows, err := tx.Query(
		`SELECT id, name, weight, prize_type, prize_value, total_stock, remaining_stock
		 FROM lottery_prizes
		 WHERE event_id = $1 AND (total_stock = 0 OR remaining_stock > 0)`,
		eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: load lottery prizes: %w", err)
	}
	defer rows.Close()

	type candidate struct {
		id, weight, totalStock, remainingStock int
		name, prizeType                        string
		prizeValue                             float64
	}
	var candidates []candidate
	var totalWeight int
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.name, &c.weight, &c.prizeType, &c.prizeValue, &c.totalStock, &c.remainingStock); err != nil {
			return nil, fmt.Errorf("store: scan prize candidate: %w", err)
		}
		candidates = append(candidates, c)
		totalWeight += c.weight
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("store: commit (no prizes): %w", err)
		}
		return &LotteryRecord{
			ID:             recordID,
			EventID:        eventID,
			UserID:         userID,
			OrderNo:        orderNo,
			RechargeAmount: rechargeAmount,
			CreatedAt:      now,
		}, nil
	}

	// Weighted random selection.
	rnd := rand.Intn(totalWeight)
	var chosen candidate
	cumulative := 0
	for _, c := range candidates {
		cumulative += c.weight
		if rnd < cumulative {
			chosen = c
			break
		}
	}

	// Decrement stock if limited.
	if chosen.totalStock > 0 {
		res, err := tx.Exec(
			`UPDATE lottery_prizes SET remaining_stock = remaining_stock - 1, updated_at = NOW()
			 WHERE id = $1 AND remaining_stock > 0`,
			chosen.id,
		)
		if err != nil {
			return nil, fmt.Errorf("store: decrement prize stock: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return nil, fmt.Errorf("store: prize stock exhausted")
		}
	}

	// Update record with prize.
	_, err = tx.Exec(
		`UPDATE lottery_records SET prize_id = $1 WHERE id = $2`,
		chosen.id, recordID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: update lottery record prize: %w", err)
	}

	// Determine actual credited amount for balance/match_recharge prizes.
	actualPrizeValue := chosen.prizeValue
	if chosen.prizeType == "match_recharge" {
		actualPrizeValue = rechargeAmount
	}

	// Credit balance for balance or match_recharge prizes.
	if (chosen.prizeType == "balance" || chosen.prizeType == "match_recharge") && actualPrizeValue > 0 {
		description := fmt.Sprintf("抽奖奖励：%s", chosen.name)
		if err := s.rechargeInTx(tx, userID, actualPrizeValue, description); err != nil {
			return nil, fmt.Errorf("store: credit lottery prize: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit lottery draw: %w", err)
	}

	return &LotteryRecord{
		ID:             recordID,
		EventID:        eventID,
		UserID:         userID,
		PrizeID:        &chosen.id,
		PrizeName:      chosen.name,
		PrizeType:      chosen.prizeType,
		PrizeValue:     actualPrizeValue,
		OrderNo:        orderNo,
		RechargeAmount: rechargeAmount,
		CreatedAt:      now,
	}, nil
}

// rechargeInTx credits balance within an existing transaction (helper for DrawLottery).
func (s *PgStore) rechargeInTx(tx *sql.Tx, userID string, amount float64, description string) error {
	_, err := tx.Exec(
		`INSERT INTO balances (user_id, balance) VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET balance = balances.balance + EXCLUDED.balance, updated_at = NOW()`,
		userID, amount,
	)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO transactions (user_id, type, amount, description, balance_after)
		 VALUES ($1, 'recharge', $2, $3, (SELECT balance FROM balances WHERE user_id = $1))`,
		userID, amount, description,
	)
	if err != nil {
		return fmt.Errorf("record transaction: %w", err)
	}
	return nil
}

// GetActiveLotteryInfo returns the active lottery event and its prizes (for user display, hides weights).
func (s *PgStore) GetActiveLotteryInfo() (*LotteryEvent, []LotteryPrize, error) {
	now := time.Now()
	var e LotteryEvent
	err := s.db.QueryRow(
		`SELECT e.id, e.name, e.description, e.status, e.min_recharge_cny, e.min_order_count_to_draw, e.start_time, e.end_time,
		        COALESCE(COUNT(r.id), 0) as participant_count, e.drawn_at, e.drawn_by, e.created_at, e.updated_at
		 FROM lottery_events e
		 LEFT JOIN lottery_records r ON e.id = r.event_id
		 WHERE e.status = 'active'
		   AND (e.start_time IS NULL OR e.start_time <= $1)
		   AND (e.end_time IS NULL OR e.end_time >= $1)
		 GROUP BY e.id
		 LIMIT 1`,
		now,
	).Scan(&e.ID, &e.Name, &e.Description, &e.Status, &e.MinRechargeCNY, &e.MinOrderCountToDraw, &e.StartTime, &e.EndTime, &e.ParticipantCount, &e.DrawnAt, &e.DrawnBy, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("store: get active lottery info: %w", err)
	}

	prizes, err := s.ListLotteryPrizes(e.ID)
	if err != nil {
		return nil, nil, err
	}

	return &e, prizes, nil
}

// DrawEventLottery performs a weighted, per-participant draw for an event.
//
// Each participant independently draws one prize according to prize weights.
// Limited-stock prizes are decremented atomically (FOR UPDATE); "none" prizes
// act as a consolation fallback so every participant gets a result. A single
// participant may win multiple prizes if the configuration includes more
// prizes than participants.
//
// Idempotency: the event row is locked (FOR UPDATE) and must be in the
// "active" state with drawn_at IS NULL. On success the event is marked
// "ended" with drawn_at/drawn_by set, preventing repeated draws.
func (s *PgStore) DrawEventLottery(eventID int, drawnBy string) ([]LotteryRecord, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Lock the event and enforce single-draw idempotency.
	var status string
	var drawnAt *time.Time
	var minOrderCountToDraw int
	err = tx.QueryRow(
		`SELECT status, drawn_at, min_order_count_to_draw FROM lottery_events WHERE id = $1 FOR UPDATE`,
		eventID,
	).Scan(&status, &drawnAt, &minOrderCountToDraw)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: lottery event not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: lock lottery event: %w", err)
	}
	if status != "active" {
		return nil, fmt.Errorf("store: event %d is not active (status=%s)", eventID, status)
	}
	if drawnAt != nil {
		return nil, fmt.Errorf("store: event %d already drawn", eventID)
	}

	// Enforce minimum qualifying-order count to draw.
	if minOrderCountToDraw > 0 {
		var cnt int
		if err := tx.QueryRow(
			`SELECT COUNT(*) FROM lottery_records WHERE event_id = $1`,
			eventID,
		).Scan(&cnt); err != nil {
			return nil, fmt.Errorf("store: count lottery records for draw threshold: %w", err)
		}
		if cnt < minOrderCountToDraw {
			return nil, fmt.Errorf("lottery: cannot draw event %d: only %d orders, need %d", eventID, cnt, minOrderCountToDraw)
		}
	}

	// Load all available prizes with stock.
	rows, err := tx.Query(
		`SELECT id, name, weight, prize_type, prize_value, total_stock, remaining_stock
		 FROM lottery_prizes
		 WHERE event_id = $1 AND (total_stock = 0 OR remaining_stock > 0)
		 ORDER BY sort_order, id`,
		eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: load prizes: %w", err)
	}
	defer rows.Close()

	type prize struct {
		id, weight, totalStock, remainingStock int
		name, prizeType                        string
		prizeValue                             float64
	}
	var prizes []prize
	var totalWeight int
	for rows.Next() {
		var p prize
		if err := rows.Scan(&p.id, &p.name, &p.weight, &p.prizeType, &p.prizeValue, &p.totalStock, &p.remainingStock); err != nil {
			return nil, fmt.Errorf("store: scan prize: %w", err)
		}
		prizes = append(prizes, p)
		totalWeight += p.weight
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(prizes) == 0 {
		return nil, fmt.Errorf("store: no prizes available for event %d", eventID)
	}
	if totalWeight <= 0 {
		return nil, fmt.Errorf("store: total prize weight must be positive for event %d", eventID)
	}

	// Load all participants (records without a prize).
	participantRows, err := tx.Query(
		`SELECT id, user_id, order_no, recharge_amount, created_at
		 FROM lottery_records
		 WHERE event_id = $1 AND prize_id IS NULL
		 ORDER BY created_at`,
		eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: load participants: %w", err)
	}
	defer participantRows.Close()

	type participant struct {
		id             int64
		userID         string
		orderNo        string
		rechargeAmount float64
		createdAt      time.Time
	}
	var participants []participant
	for participantRows.Next() {
		var p participant
		if err := participantRows.Scan(&p.id, &p.userID, &p.orderNo, &p.rechargeAmount, &p.createdAt); err != nil {
			return nil, fmt.Errorf("store: scan participant: %w", err)
		}
		participants = append(participants, p)
	}
	if err := participantRows.Err(); err != nil {
		return nil, err
	}
	if len(participants) == 0 {
		return nil, fmt.Errorf("store: no participants found for event %d", eventID)
	}

	// Weighted independent draw: each participant draws one prize.
	var winners []LotteryRecord
	type candidate struct {
		prize
		weight int
	}

	// pickPrize builds the candidate pool from the current prize stock state
	// and returns a weighted-random choice (falling back to "none" prizes when
	// all stock prizes are exhausted). Returns false if no prize is available.
	pickPrize := func() (prize, bool) {
		var pool []candidate
		var poolWeight int
		var nonePrizes []candidate
		for _, prz := range prizes {
			if prz.prizeType == "none" {
				nonePrizes = append(nonePrizes, candidate{prz, prz.weight})
				continue
			}
			if prz.totalStock > 0 && prz.remainingStock <= 0 {
				continue
			}
			pool = append(pool, candidate{prz, prz.weight})
			poolWeight += prz.weight
		}
		if poolWeight > 0 {
			rnd := rand.Intn(poolWeight)
			cumulative := 0
			for _, c := range pool {
				cumulative += c.weight
				if rnd < cumulative {
					return c.prize, true
				}
			}
			// Fallback (should be unreachable): last pool element.
			return pool[len(pool)-1].prize, true
		}
		if len(nonePrizes) > 0 {
			return nonePrizes[rand.Intn(len(nonePrizes))].prize, true
		}
		return prize{}, false
	}

	// markSoldOut zeroes the cached stock for a prize id so later draws and
	// retries skip it.
	markSoldOut := func(prizeID int) {
		for i := range prizes {
			if prizes[i].id == prizeID {
				prizes[i].remainingStock = 0
			}
		}
	}

	for _, pt := range participants {
		// Retry the same participant when a limited-stock prize is claimed by a
		// concurrent draw within the transaction (rare, since the whole draw is
		// serialized by the event row lock, but the UPDATE ... WHERE remaining
		// _stock > 0 guard still protects against logical races). Cap retries
		// to the number of distinct prizes to guarantee termination.
		var chosen prize
		var ok bool
		for attempt := 0; attempt < len(prizes); attempt++ {
			chosen, ok = pickPrize()
			if !ok {
				break // no prizes available at all
			}

			// Decrement stock for limited prizes atomically.
			if chosen.totalStock > 0 {
				res, err := tx.Exec(
					`UPDATE lottery_prizes SET remaining_stock = remaining_stock - 1, updated_at = NOW()
					 WHERE id = $1 AND remaining_stock > 0`,
					chosen.id,
				)
				if err != nil {
					return nil, fmt.Errorf("store: decrement prize stock: %w", err)
				}
				n, _ := res.RowsAffected()
				if n == 0 {
					// Lost the last unit — mark it sold out in cache and retry.
					markSoldOut(chosen.id)
					continue
				}
				// Reflect the successful decrement in the cached slice so later
				// participants see the updated stock.
				for i := range prizes {
					if prizes[i].id == chosen.id {
						prizes[i].remainingStock--
					}
				}
			}
			ok = true
			break
		}
		if !ok {
			// No prize available for this participant (all stock exhausted and
			// no "none" consolation configured) — leave them without a prize.
			continue
		}

		// Update participant record with prize.
		_, err := tx.Exec(
			`UPDATE lottery_records SET prize_id = $1 WHERE id = $2`,
			chosen.id, pt.id,
		)
		if err != nil {
			return nil, fmt.Errorf("store: update lottery record: %w", err)
		}

		// Determine actual prize value.
		actualPrizeValue := chosen.prizeValue
		if chosen.prizeType == "match_recharge" {
			actualPrizeValue = pt.rechargeAmount
		}

		// Credit balance for balance/match_recharge prizes.
		if (chosen.prizeType == "balance" || chosen.prizeType == "match_recharge") && actualPrizeValue > 0 {
			description := fmt.Sprintf("抽奖奖励：%s", chosen.name)
			if err := s.rechargeInTx(tx, pt.userID, actualPrizeValue, description); err != nil {
				return nil, fmt.Errorf("store: credit prize: %w", err)
			}
		}

		winners = append(winners, LotteryRecord{
			ID:             pt.id,
			EventID:        eventID,
			UserID:         pt.userID,
			PrizeID:        &chosen.id,
			PrizeName:      chosen.name,
			PrizeType:      chosen.prizeType,
			PrizeValue:     actualPrizeValue,
			OrderNo:        pt.orderNo,
			RechargeAmount: pt.rechargeAmount,
			CreatedAt:      pt.createdAt,
		})
	}

	// Mark the event as drawn + ended (idempotent guard).
	var drawnByArg interface{} = drawnBy
	if drawnBy == "" {
		drawnByArg = nil
	}
	if _, err := tx.Exec(
		`UPDATE lottery_events
		 SET status = 'ended', drawn_at = NOW(), drawn_by = $1, updated_at = NOW()
		 WHERE id = $2 AND status = 'active' AND drawn_at IS NULL`,
		drawnByArg, eventID,
	); err != nil {
		return nil, fmt.Errorf("store: mark event drawn: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit lottery draw: %w", err)
	}

	return winners, nil
}
