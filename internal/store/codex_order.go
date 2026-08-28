package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

type CodexOrder struct {
	ID             string           `json:"id"`
	OrderNo        string           `json:"order_no"`
	ProductID      int              `json:"product_id"`
	Product        *CodexProduct    `json:"product,omitempty"`
	UserID         *string          `json:"user_id,omitempty"`
	GuestContact   json.RawMessage  `json:"guest_contact,omitempty"`
	Amount         float64          `json:"amount"`
	PayMethod      string           `json:"pay_method"`
	Status         string           `json:"status"`
	PayTime        *time.Time       `json:"pay_time,omitempty"`
	CallbackData   *json.RawMessage `json:"callback_data,omitempty"`
	RedemptionCode *string          `json:"redemption_code,omitempty"`
	ShippedAt      *time.Time       `json:"shipped_at,omitempty"`
	ShippedBy      *string          `json:"shipped_by,omitempty"`
	ServiceWechat  string           `json:"service_wechat"`
	CreatedAt      time.Time        `json:"created_at"`
	ExpiredAt      time.Time        `json:"expired_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type AdminCodexOrder struct {
	CodexOrder
	ContactInfo string `json:"contact_info"`
}

// generateCodexOrderNo 生成 Codex 订单号（CDX + 时间戳14位 + 随机6位）
func generateCodexOrderNo() string {
	now := time.Now()
	return fmt.Sprintf("CDX%s%06d", now.Format("20060102150405"), rand.Intn(1000000))
}

// CreateCodexOrder 创建 Codex 订单（支持游客和注册用户）
func (s *PgStore) CreateCodexOrder(productID int, userID *string, guestContact json.RawMessage, serviceWechat string) (*CodexOrder, error) {
	if userID == nil && len(guestContact) == 0 {
		return nil, fmt.Errorf("store: codex order requires user_id or guest_contact")
	}

	product, err := s.GetCodexProductByID(productID)
	if err != nil {
		return nil, fmt.Errorf("store: product not found: %w", err)
	}
	if product.Status != "active" {
		return nil, fmt.Errorf("store: product is not available")
	}

	orderNo := generateCodexOrderNo()
	expiredAt := time.Now().Add(15 * time.Minute)

	if serviceWechat == "" {
		serviceWechat = "codex-service-01"
	}

	var order CodexOrder
	err = s.db.QueryRow(`
		INSERT INTO codex_orders (order_no, product_id, user_id, guest_contact, amount, expired_at, service_wechat)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, order_no, product_id, user_id, guest_contact, amount, pay_method, status,
		          pay_time, callback_data, redemption_code, shipped_at, shipped_by, service_wechat,
		          created_at, expired_at, updated_at
	`, orderNo, productID, userID, guestContact, product.PriceCNY, expiredAt, serviceWechat).Scan(
		&order.ID, &order.OrderNo, &order.ProductID, &order.UserID, &order.GuestContact, &order.Amount,
		&order.PayMethod, &order.Status, &order.PayTime, &order.CallbackData, &order.RedemptionCode,
		&order.ShippedAt, &order.ShippedBy, &order.ServiceWechat, &order.CreatedAt, &order.ExpiredAt, &order.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: create codex order: %w", err)
	}

	order.Product = product
	return &order, nil
}

// GetCodexOrderByNo 根据订单号查询（公开接口，游客可查）
func (s *PgStore) GetCodexOrderByNo(orderNo string) (*CodexOrder, error) {
	var order CodexOrder
	var product CodexProduct

	err := s.db.QueryRow(`
		SELECT o.id, o.order_no, o.product_id, o.user_id, o.guest_contact, o.amount, o.pay_method, o.status,
		       o.pay_time, o.callback_data, o.redemption_code, o.shipped_at, o.shipped_by, o.service_wechat,
		       o.created_at, o.expired_at, o.updated_at,
		       p.id, p.sku, p.name, p.description, p.price_cny, p.sort_order, p.status, p.created_at, p.updated_at
		FROM codex_orders o
		JOIN codex_products p ON p.id = o.product_id
		WHERE o.order_no = $1
	`, orderNo).Scan(
		&order.ID, &order.OrderNo, &order.ProductID, &order.UserID, &order.GuestContact, &order.Amount,
		&order.PayMethod, &order.Status, &order.PayTime, &order.CallbackData, &order.RedemptionCode,
		&order.ShippedAt, &order.ShippedBy, &order.ServiceWechat, &order.CreatedAt, &order.ExpiredAt, &order.UpdatedAt,
		&product.ID, &product.SKU, &product.Name, &product.Description, &product.PriceCNY,
		&product.SortOrder, &product.Status, &product.CreatedAt, &product.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: codex order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get codex order: %w", err)
	}

	order.Product = &product
	return &order, nil
}

// MarkCodexOrderPaid 标记订单已支付（幂等操作）
func (s *PgStore) MarkCodexOrderPaid(orderNo string, callbackData []byte) error {
	result, err := s.db.Exec(`
		UPDATE codex_orders
		SET status = 'paid', pay_time = NOW(), callback_data = $1, updated_at = NOW()
		WHERE order_no = $2 AND status IN ('pending', 'expired')
	`, callbackData, orderNo)
	if err != nil {
		return fmt.Errorf("store: mark codex order paid: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("store: codex order not updatable or already paid")
	}
	return nil
}

// ShipCodexOrder 管理员发货（填写兑换码）
func (s *PgStore) ShipCodexOrder(orderNo, redemptionCode, adminUserID string) error {
	result, err := s.db.Exec(`
		UPDATE codex_orders
		SET status = 'shipped', redemption_code = $1, shipped_at = NOW(), shipped_by = $2, updated_at = NOW()
		WHERE order_no = $3 AND status = 'paid'
	`, redemptionCode, adminUserID, orderNo)
	if err != nil {
		return fmt.Errorf("store: ship codex order: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("store: codex order not found or not paid")
	}
	return nil
}

// ListAllCodexOrders 管理后台订单列表（支持状态筛选）
func (s *PgStore) ListAllCodexOrders(limit, offset int, status string) ([]AdminCodexOrder, int, error) {
	var total int
	var args []interface{}
	where := ""
	argIndex := 1

	if status != "" {
		where = fmt.Sprintf(" WHERE o.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	countSQL := "SELECT COUNT(*) FROM codex_orders o" + where
	if err := s.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count codex orders: %w", err)
	}

	selectSQL := fmt.Sprintf(`
		SELECT o.id, o.order_no, o.product_id, o.user_id, o.guest_contact, o.amount, o.pay_method, o.status,
		       o.pay_time, o.callback_data, o.redemption_code, o.shipped_at, o.shipped_by, o.service_wechat,
		       o.created_at, o.expired_at, o.updated_at,
		       p.sku, p.name,
		       COALESCE(u.phone, u.email,
		           CASE
		             WHEN o.guest_contact IS NULL THEN '游客'
		             -- 纯字符串联系方式（前端直接传字符串）：原样去除引号输出
		             WHEN jsonb_typeof(o.guest_contact) = 'string' THEN o.guest_contact #>> '{}'
		             -- 旧的对象格式：取 phone/email/wechat 任一
		             ELSE COALESCE(o.guest_contact->>'phone', o.guest_contact->>'email', o.guest_contact->>'wechat', '游客')
		           END)
		FROM codex_orders o
		JOIN codex_products p ON p.id = o.product_id
		LEFT JOIN users u ON u.id = o.user_id`+where+
		` ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d`, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := s.db.Query(selectSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list codex orders: %w", err)
	}
	defer rows.Close()

	var orders []AdminCodexOrder
	for rows.Next() {
		var ao AdminCodexOrder
		var productSKU, productName string

		if err := rows.Scan(
			&ao.ID, &ao.OrderNo, &ao.ProductID, &ao.UserID, &ao.GuestContact, &ao.Amount,
			&ao.PayMethod, &ao.Status, &ao.PayTime, &ao.CallbackData, &ao.RedemptionCode,
			&ao.ShippedAt, &ao.ShippedBy, &ao.ServiceWechat, &ao.CreatedAt, &ao.ExpiredAt, &ao.UpdatedAt,
			&productSKU, &productName, &ao.ContactInfo,
		); err != nil {
			return nil, 0, fmt.Errorf("store: scan codex order: %w", err)
		}

		ao.Product = &CodexProduct{
			SKU:  productSKU,
			Name: productName,
		}
		orders = append(orders, ao)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: iterate codex orders: %w", err)
	}

	return orders, total, nil
}

// CodexRefundResult records the outcome of an admin-initiated Codex refund.
type CodexRefundResult struct {
	OutRequestNo  string
	AlipayTradeNo string
	Amount        float64
}

// RecordCodexRefund records a refund attempt for a Codex order and, when
// succeeded is true, atomically marks the order as refunded. This keeps a
// full audit trail in codex_refunds (required by migration 000125) that the
// previous direct-UPDATE implementation left empty.
func (s *PgStore) RecordCodexRefund(orderNo, outRequestNo, reason string, amount float64, operatorID string, succeeded bool, alipayTradeNo, errMsg string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: record codex refund: begin: %w", err)
	}
	defer tx.Rollback()

	status := "pending"
	if succeeded {
		status = "succeeded"
	} else if errMsg != "" {
		status = "failed"
	}

	_, err = tx.Exec(`
		INSERT INTO codex_refunds (codex_order_no, amount, reason, status, out_request_no, alipay_trade_no, operator_id, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, orderNo, amount, reason, status, outRequestNo, alipayTradeNo, operatorID, errMsg)
	if err != nil {
		return fmt.Errorf("store: record codex refund: insert: %w", err)
	}

	if succeeded {
		_, err = tx.Exec(`
			UPDATE codex_orders
			SET status = 'refunded', updated_at = NOW()
			WHERE order_no = $1
		`, orderNo)
		if err != nil {
			return fmt.Errorf("store: record codex refund: mark order: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: record codex refund: commit: %w", err)
	}
	return nil
}

// ExpireCodexOrders 将过期的待支付订单标记为 expired
func (s *PgStore) ExpireCodexOrders() (int, error) {
	result, err := s.db.Exec(`
		UPDATE codex_orders
		SET status = 'expired', updated_at = NOW()
		WHERE status = 'pending' AND expired_at < NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("store: expire codex orders: %w", err)
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}
