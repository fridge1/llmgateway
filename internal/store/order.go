package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// Order represents a payment order.
type Order struct {
	ID                 string           `json:"id"`
	UserID             string           `json:"user_id"`
	TenantID           *string          `json:"tenant_id,omitempty"`
	OrderNo            string           `json:"order_no"`
	Amount             float64          `json:"amount"`
	PayMethod          string           `json:"pay_method"`
	Status             string           `json:"status"`
	PayTime            *time.Time       `json:"pay_time,omitempty"`
	CallbackData       *json.RawMessage `json:"callback_data,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	ExpiredAt          time.Time        `json:"expired_at"`
	SubscriptionPlanID *int             `json:"subscription_plan_id,omitempty"`
}

// AdminOrder enriches Order with user info for admin views.
type AdminOrder struct {
	Order
	UserIdentifier string `json:"user_identifier"`
}

// generateOrderNo creates an order number with format yyyyMMddHHmmss + 6 random digits.
func generateOrderNo() string {
	now := time.Now()
	return fmt.Sprintf("%s%06d", now.Format("20060102150405"), rand.Intn(1000000))
}

// CreateOrder creates a new payment order for the given user and amount.
// If tenantID is provided, the order is associated with that tenant.
func (s *PgStore) CreateOrder(userID string, amount float64, tenantID *string) (*Order, error) {
	return s.CreateOrderWithPlan(userID, amount, tenantID, nil)
}

// CreateOrderWithPlan creates a payment order optionally linked to a subscription plan.
func (s *PgStore) CreateOrderWithPlan(userID string, amount float64, tenantID *string, planID *int) (*Order, error) {
	if amount < 0.01 {
		return nil, fmt.Errorf("store: order amount must be at least 0.01")
	}
	orderNo := generateOrderNo()
	expiredAt := time.Now().Add(15 * time.Minute)

	var o Order
	err := s.db.QueryRow(
		`INSERT INTO orders (user_id, order_no, amount, expired_at, tenant_id, subscription_plan_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, order_no, amount, pay_method, status, pay_time, callback_data, created_at, expired_at, tenant_id, subscription_plan_id`,
		userID, orderNo, amount, expiredAt, tenantID, planID,
	).Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Amount, &o.PayMethod, &o.Status,
		&o.PayTime, &o.CallbackData, &o.CreatedAt, &o.ExpiredAt, &o.TenantID, &o.SubscriptionPlanID)
	if err != nil {
		return nil, fmt.Errorf("store: create order: %w", err)
	}
	return &o, nil
}

// GetOrderByNo retrieves an order by its order number.
func (s *PgStore) GetOrderByNo(orderNo string) (*Order, error) {
	var o Order
	err := s.db.QueryRow(
		`SELECT id, user_id, order_no, amount, pay_method, status, pay_time, callback_data, created_at, expired_at, tenant_id, subscription_plan_id
		 FROM orders WHERE order_no = $1`,
		orderNo,
	).Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Amount, &o.PayMethod, &o.Status,
		&o.PayTime, &o.CallbackData, &o.CreatedAt, &o.ExpiredAt, &o.TenantID, &o.SubscriptionPlanID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: order not found: %s", orderNo)
	}
	if err != nil {
		return nil, fmt.Errorf("store: get order: %w", err)
	}
	return &o, nil
}

// FulfillAlipayPaidOrder marks the order paid and credits balance atomically.
// If the order is already paid (e.g. retry), it only inserts the recharge when the matching
// transaction row is missing — fixes partial failure where MarkOrderPaid succeeded but Recharge did not.
func (s *PgStore) FulfillAlipayPaidOrder(orderNo string, callbackData []byte) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: fulfill alipay order: begin: %w", err)
	}
	defer tx.Rollback()

	var userID string
	var amount float64
	var status string
	var tenantID *string
	var subscriptionPlanID *int
	err = tx.QueryRow(
		`SELECT user_id, amount, status, tenant_id, subscription_plan_id FROM orders WHERE order_no = $1 FOR UPDATE`,
		orderNo,
	).Scan(&userID, &amount, &status, &tenantID, &subscriptionPlanID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("store: order not found: %s", orderNo)
	}
	if err != nil {
		return fmt.Errorf("store: fulfill alipay order: select: %w", err)
	}

	desc := fmt.Sprintf("Alipay recharge order %s", orderNo)

	if status == "pending" || status == "expired" {
		_, err = tx.Exec(
			`UPDATE orders SET status = 'paid', pay_time = NOW(), callback_data = $1
			 WHERE order_no = $2`,
			callbackData, orderNo,
		)
		if err != nil {
			return fmt.Errorf("store: fulfill alipay order: mark paid: %w", err)
		}
	} else if status == "paid" {
		var cnt int
		err = tx.QueryRow(
			`SELECT COUNT(*) FROM transactions WHERE user_id = $1 AND type = 'recharge' AND description = $2`,
			userID, desc,
		).Scan(&cnt)
		if err != nil {
			return fmt.Errorf("store: fulfill alipay order: check tx: %w", err)
		}
		if cnt > 0 {
			return tx.Commit()
		}
		// paid but missing recharge row — complete credit on Alipay retry
	} else {
		return fmt.Errorf("store: order %s unexpected status %s", orderNo, status)
	}

	// Always credit to user balance (tenant_id is just for record keeping)
	res, err := tx.Exec(
		`UPDATE balances SET balance = balance + $1, updated_at = NOW() WHERE user_id = $2`,
		amount, userID,
	)
	if err != nil {
		return fmt.Errorf("store: fulfill alipay order: recharge: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: fulfill alipay order: no balance row for user %s", userID)
	}

	var balanceAfter float64
	err = tx.QueryRow(`SELECT balance FROM balances WHERE user_id = $1`, userID).Scan(&balanceAfter)
	if err != nil {
		return fmt.Errorf("store: fulfill alipay order: get balance: %w", err)
	}
	_, err = tx.Exec(
		`INSERT INTO transactions (user_id, type, amount, balance_after, description)
		 VALUES ($1, 'recharge', $2, $3, $4)`,
		userID, amount, balanceAfter, desc,
	)
	if err != nil {
		return fmt.Errorf("store: fulfill alipay order: insert tx: %w", err)
	}
	return tx.Commit()
}

// MarkOrderPaid transitions an order to paid and records callback data.
// Accepts pending or expired: async notify may arrive after local 15m expiry window.
func (s *PgStore) MarkOrderPaid(orderNo string, callbackData []byte) error {
	result, err := s.db.Exec(
		`UPDATE orders SET status = 'paid', pay_time = NOW(), callback_data = $1
		 WHERE order_no = $2 AND status IN ('pending', 'expired')`,
		callbackData, orderNo,
	)
	if err != nil {
		return fmt.Errorf("store: mark order paid: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("store: order %s not updatable (not pending/expired or unknown order_no)", orderNo)
	}
	return nil
}

// OrderStatusCounts holds aggregated order counts by status.
type OrderStatusCounts struct {
	Paid    int `json:"paid"`
	Pending int `json:"pending"`
	Expired int `json:"expired"`
}

// ListOrders returns paginated orders for the given user, newest first, along with status counts.
func (s *PgStore) ListOrders(userID string, limit, offset int) ([]Order, int, *OrderStatusCounts, error) {
	var total int
	counts := &OrderStatusCounts{}

	err := s.db.QueryRow(
		`SELECT COUNT(*),
		        COUNT(*) FILTER (WHERE status = 'paid'),
		        COUNT(*) FILTER (WHERE status = 'pending'),
		        COUNT(*) FILTER (WHERE status = 'expired')
		 FROM orders WHERE user_id = $1`, userID,
	).Scan(&total, &counts.Paid, &counts.Pending, &counts.Expired)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("store: count orders: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT id, user_id, order_no, amount, pay_method, status, pay_time, callback_data, created_at, expired_at, tenant_id, subscription_plan_id
		 FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("store: list orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Amount, &o.PayMethod, &o.Status,
			&o.PayTime, &o.CallbackData, &o.CreatedAt, &o.ExpiredAt, &o.TenantID, &o.SubscriptionPlanID); err != nil {
			return nil, 0, nil, fmt.Errorf("store: scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, total, counts, nil
}

// ExpireOrders marks pending orders as expired if their expiry time has passed.
func (s *PgStore) ExpireOrders() (int, error) {
	result, err := s.db.Exec(
		`UPDATE orders SET status = 'expired' WHERE status = 'pending' AND expired_at < NOW()`,
	)
	if err != nil {
		return 0, fmt.Errorf("store: expire orders: %w", err)
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// ListAllOrders returns paginated orders across all users, newest first.
// If status is non-empty, only orders with that status are returned.
func (s *PgStore) ListAllOrders(limit, offset int, status string) ([]AdminOrder, int, error) {
	var total int
	var args []any
	where := ""
	if status != "" {
		where = " WHERE o.status = $1"
		args = append(args, status)
	}

	countSQL := `SELECT COUNT(*) FROM orders o` + where
	if err := s.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count all orders: %w", err)
	}

	selectSQL := `SELECT o.id, o.user_id, o.order_no, o.amount, o.pay_method, o.status,
		        o.pay_time, o.callback_data, o.created_at, o.expired_at, o.tenant_id, o.subscription_plan_id,
		        COALESCE(u.phone, u.email, NULLIF(u.nickname, ''), SUBSTRING(u.id::text, 1, 8))
		 FROM orders o
		 LEFT JOIN users u ON u.id = o.user_id` + where +
		fmt.Sprintf(` ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.Query(selectSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list all orders: %w", err)
	}
	defer rows.Close()

	var orders []AdminOrder
	for rows.Next() {
		var ao AdminOrder
		if err := rows.Scan(&ao.ID, &ao.UserID, &ao.OrderNo, &ao.Amount, &ao.PayMethod, &ao.Status,
			&ao.PayTime, &ao.CallbackData, &ao.CreatedAt, &ao.ExpiredAt, &ao.TenantID, &ao.SubscriptionPlanID, &ao.UserIdentifier); err != nil {
			return nil, 0, fmt.Errorf("store: scan admin order: %w", err)
		}
		orders = append(orders, ao)
	}
	return orders, total, nil
}
