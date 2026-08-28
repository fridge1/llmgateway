package store

import (
	"errors"
	"fmt"
	"time"
)

// ErrInsufficientBalanceForRefund is returned when the user's available
// balance cannot cover the clawback for a refund.
var ErrInsufficientBalanceForRefund = errors.New("user balance insufficient for refund clawback")

// ErrRefundExceedsOrder is returned when accumulated refunds would exceed the
// order's paid amount.
var ErrRefundExceedsOrder = errors.New("refund amount exceeds order remaining refundable amount")

// Refund records an admin-initiated Alipay refund.
type Refund struct {
	ID             string    `json:"id"`
	OrderNo        string    `json:"order_no"`
	UserID         string    `json:"user_id"`
	UserIdentifier string    `json:"user_identifier,omitempty"`
	Amount         float64   `json:"amount"`
	Reason         string    `json:"reason"`
	Status         string    `json:"status"`
	OutRequestNo   string    `json:"out_request_no"`
	AlipayTradeNo  *string   `json:"alipay_trade_no"`
	OperatorID     *string   `json:"operator_id"`
	ErrorMessage   *string   `json:"error_message"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

const refundCols = `id, order_no, user_id, amount, reason, status, out_request_no, alipay_trade_no, operator_id, error_message, created_at, updated_at`

// CreateRefund validates refundability and inserts a pending refund record.
// It checks: order exists & paid & belongs to the stated user; accumulated
// refunds (pending+success) don't exceed the paid amount; and the user's
// available balance covers the clawback.
func (s *PgStore) CreateRefund(orderNo, operatorID string, amount float64, reason string) (*Refund, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: refund begin: %w", err)
	}
	defer tx.Rollback()

	var userID string
	var orderAmount float64
	var status string
	err = tx.QueryRow(
		`SELECT user_id, amount, status FROM orders WHERE order_no=$1 FOR UPDATE`, orderNo,
	).Scan(&userID, &orderAmount, &status)
	if err != nil {
		return nil, fmt.Errorf("store: refund order lookup: %w", err)
	}
	if status != "paid" {
		return nil, fmt.Errorf("order %s is not paid (status=%s)", orderNo, status)
	}

	var refunded float64
	if err := tx.QueryRow(
		`SELECT COALESCE(SUM(amount),0) FROM refunds WHERE order_no=$1 AND status IN ('pending','success')`,
		orderNo,
	).Scan(&refunded); err != nil {
		return nil, fmt.Errorf("store: refund sum: %w", err)
	}
	if refunded+amount > orderAmount+1e-9 {
		return nil, ErrRefundExceedsOrder
	}

	// Balance clawback feasibility: available (non-frozen) balance must cover it.
	var available float64
	if err := tx.QueryRow(
		`SELECT balance FROM balances WHERE user_id=$1 FOR UPDATE`, userID,
	).Scan(&available); err != nil {
		return nil, fmt.Errorf("store: refund balance lookup: %w", err)
	}
	if available < amount {
		return nil, ErrInsufficientBalanceForRefund
	}

	outRequestNo := fmt.Sprintf("rf_%s_%d", orderNo, time.Now().UnixNano())
	var rf Refund
	err = tx.QueryRow(
		`INSERT INTO refunds (order_no, user_id, amount, reason, out_request_no, operator_id)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING `+refundCols,
		orderNo, userID, amount, reason, outRequestNo, operatorID,
	).Scan(&rf.ID, &rf.OrderNo, &rf.UserID, &rf.Amount, &rf.Reason, &rf.Status, &rf.OutRequestNo,
		&rf.AlipayTradeNo, &rf.OperatorID, &rf.ErrorMessage, &rf.CreatedAt, &rf.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: refund insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: refund commit: %w", err)
	}
	return &rf, nil
}

// CompleteRefund marks a pending refund successful and claws the amount back
// from the user's balance with a negative 'refund' transaction — all in one
// DB transaction so money never goes missing between states.
func (s *PgStore) CompleteRefund(refundID, alipayTradeNo string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: complete refund begin: %w", err)
	}
	defer tx.Rollback()

	var userID string
	var amount float64
	err = tx.QueryRow(
		`UPDATE refunds SET status='success', alipay_trade_no=$2, updated_at=now()
		 WHERE id=$1 AND status='pending'
		 RETURNING user_id, amount`, refundID, alipayTradeNo,
	).Scan(&userID, &amount)
	if err != nil {
		return fmt.Errorf("store: complete refund update: %w", err)
	}

	var balanceAfter float64
	err = tx.QueryRow(
		`UPDATE balances SET balance = balance - $1, updated_at = now()
		 WHERE user_id = $2 RETURNING balance`, amount, userID,
	).Scan(&balanceAfter)
	if err != nil {
		return fmt.Errorf("store: complete refund clawback: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO transactions (user_id, type, amount, balance_after, description)
		 VALUES ($1, 'refund', $2, $3, $4)`,
		userID, -amount, balanceAfter, "充值退款（退回支付宝）",
	); err != nil {
		return fmt.Errorf("store: complete refund transaction: %w", err)
	}

	return tx.Commit()
}

// FailRefund marks a pending refund failed with the upstream error message.
func (s *PgStore) FailRefund(refundID, errorMessage string) error {
	res, err := s.db.Exec(
		`UPDATE refunds SET status='failed', error_message=$2, updated_at=now()
		 WHERE id=$1 AND status='pending'`, refundID, errorMessage)
	if err != nil {
		return fmt.Errorf("store: fail refund: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("refund not found or not pending")
	}
	return nil
}

// ListRefunds returns refunds newest-first with user identifier for the admin view.
func (s *PgStore) ListRefunds(limit, offset int) ([]Refund, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count refunds: %w", err)
	}
	rows, err := s.db.Query(
		`SELECT r.id, r.order_no, r.user_id, r.amount, r.reason, r.status, r.out_request_no,
		        r.alipay_trade_no, r.operator_id, r.error_message, r.created_at, r.updated_at,
		        COALESCE(u.phone, u.email, NULLIF(u.nickname, ''), SUBSTRING(u.id::text, 1, 8))
		 FROM refunds r JOIN users u ON u.id = r.user_id
		 ORDER BY r.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list refunds: %w", err)
	}
	defer rows.Close()
	var list []Refund
	for rows.Next() {
		var rf Refund
		if err := rows.Scan(&rf.ID, &rf.OrderNo, &rf.UserID, &rf.Amount, &rf.Reason, &rf.Status, &rf.OutRequestNo,
			&rf.AlipayTradeNo, &rf.OperatorID, &rf.ErrorMessage, &rf.CreatedAt, &rf.UpdatedAt, &rf.UserIdentifier); err != nil {
			return nil, 0, fmt.Errorf("store: scan refund: %w", err)
		}
		list = append(list, rf)
	}
	return list, total, nil
}
