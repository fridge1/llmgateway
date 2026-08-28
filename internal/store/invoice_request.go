package store

import (
	"database/sql"
	"fmt"
	"time"
)

// InvoiceRequest represents an invoice request.
type InvoiceRequest struct {
	ID              int64      `json:"id"`
	UserID          string     `json:"user_id"`
	TitleID         int64      `json:"title_id"`
	InvoiceType     string     `json:"invoice_type"` // "normal" | "special"
	TotalAmount     float64    `json:"total_amount"`
	Status          string     `json:"status"` // "pending" | "processing" | "completed" | "rejected" | "cancelled"
	Remark          string     `json:"remark"`
	RejectReason    string     `json:"reject_reason"`
	InvoiceFilePath string     `json:"invoice_file_path"`
	InvoiceNumber   string     `json:"invoice_number"`
	RiskLevel       string     `json:"risk_level"`   // '' | auto_ok | needs_review
	RiskReasons     string     `json:"risk_reasons"` // ；-joined rule hits
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// InvoiceRequestOrder links a request to an order.
type InvoiceRequestOrder struct {
	ID        int64   `json:"id"`
	RequestID int64   `json:"request_id"`
	OrderID   string  `json:"order_id"`
	Amount    float64 `json:"amount"`
}

// InvoiceRequestDetail is a request with its title and orders for admin view.
type InvoiceRequestDetail struct {
	InvoiceRequest
	Title  InvoiceTitle          `json:"title"`
	Orders []InvoiceRequestOrder `json:"orders"`
	// Admin view extras
	UserIdentifier string `json:"user_identifier,omitempty"`
}

const invoiceRequestColumns = `id, user_id, title_id, invoice_type, total_amount, status, remark, reject_reason, invoice_file_path, invoice_number, risk_level, risk_reasons, completed_at, created_at, updated_at`

func scanInvoiceRequest(row interface{ Scan(...any) error }) (*InvoiceRequest, error) {
	var r InvoiceRequest
	err := row.Scan(&r.ID, &r.UserID, &r.TitleID, &r.InvoiceType, &r.TotalAmount,
		&r.Status, &r.Remark, &r.RejectReason, &r.InvoiceFilePath, &r.InvoiceNumber,
		&r.RiskLevel, &r.RiskReasons, &r.CompletedAt, &r.CreatedAt, &r.UpdatedAt)
	return &r, err
}

// ListAvailableOrders returns paid orders not tied to active invoice requests.
func (s *PgStore) ListAvailableOrders(userID string) ([]Order, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, order_no, amount, pay_method, status, pay_time, callback_data, created_at, expired_at
		 FROM orders
		 WHERE user_id = $1 AND status = 'paid'
		   AND NOT EXISTS (
		     SELECT 1 FROM invoice_request_orders iro
		     JOIN invoice_requests ir ON ir.id = iro.request_id
		     WHERE iro.order_id = orders.id AND ir.status NOT IN ('rejected', 'cancelled')
		   )
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list available orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Amount, &o.PayMethod, &o.Status,
			&o.PayTime, &o.CallbackData, &o.CreatedAt, &o.ExpiredAt); err != nil {
			return nil, fmt.Errorf("store: scan order: %w", err)
		}
		orders = append(orders, o)
	}
	if orders == nil {
		orders = []Order{}
	}
	return orders, nil
}

// CreateInvoiceRequest creates a request and links it to the given order IDs atomically.
func (s *PgStore) CreateInvoiceRequest(userID string, titleID int64, invoiceType, remark string, orderIDs []string) (*InvoiceRequest, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Verify all orders belong to user, are paid, and not already linked to active requests.
	var totalAmount float64
	type orderAmount struct {
		id     string
		amount float64
	}
	var oas []orderAmount
	for _, oid := range orderIDs {
		var oa orderAmount
		err := tx.QueryRow(
			`SELECT id, amount FROM orders WHERE id=$1 AND user_id=$2 AND status='paid'`, oid, userID,
		).Scan(&oa.id, &oa.amount)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("store: order %s not found or not paid", oid)
		}
		if err != nil {
			return nil, fmt.Errorf("store: verify order: %w", err)
		}

		// Check not already in active request
		var count int
		err = tx.QueryRow(
			`SELECT COUNT(*) FROM invoice_request_orders iro
			 JOIN invoice_requests ir ON ir.id = iro.request_id
			 WHERE iro.order_id = $1 AND ir.status NOT IN ('rejected', 'cancelled')`, oid,
		).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("store: check duplicate: %w", err)
		}
		if count > 0 {
			return nil, fmt.Errorf("store: order %s already has an active invoice request", oid)
		}

		oas = append(oas, oa)
		totalAmount += oa.amount
	}

	// Insert request
	var req InvoiceRequest
	err = tx.QueryRow(
		`INSERT INTO invoice_requests (user_id, title_id, invoice_type, total_amount, remark)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+invoiceRequestColumns,
		userID, titleID, invoiceType, totalAmount, remark,
	).Scan(&req.ID, &req.UserID, &req.TitleID, &req.InvoiceType, &req.TotalAmount,
		&req.Status, &req.Remark, &req.RejectReason, &req.InvoiceFilePath, &req.InvoiceNumber,
		&req.RiskLevel, &req.RiskReasons, &req.CompletedAt, &req.CreatedAt, &req.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: insert invoice request: %w", err)
	}

	// Link orders
	for _, oa := range oas {
		_, err := tx.Exec(
			`INSERT INTO invoice_request_orders (request_id, order_id, amount) VALUES ($1, $2, $3)`,
			req.ID, oa.id, oa.amount,
		)
		if err != nil {
			return nil, fmt.Errorf("store: link order: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit: %w", err)
	}
	return &req, nil
}

// ListInvoiceRequests returns paginated requests for a user.
func (s *PgStore) ListInvoiceRequests(userID string, limit, offset int) ([]InvoiceRequest, int, error) {
	var total int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM invoice_requests WHERE user_id=$1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("store: count invoice requests: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT `+invoiceRequestColumns+` FROM invoice_requests WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list invoice requests: %w", err)
	}
	defer rows.Close()

	var requests []InvoiceRequest
	for rows.Next() {
		r, err := scanInvoiceRequest(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("store: scan invoice request: %w", err)
		}
		requests = append(requests, *r)
	}
	if requests == nil {
		requests = []InvoiceRequest{}
	}
	return requests, total, nil
}

// GetInvoiceRequest returns a single request. If userID is empty, no user filter (admin).
func (s *PgStore) GetInvoiceRequest(id int64, userID string) (*InvoiceRequest, error) {
	query := `SELECT ` + invoiceRequestColumns + ` FROM invoice_requests WHERE id=$1`
	var row *sql.Row
	if userID != "" {
		query += ` AND user_id=$2`
		row = s.db.QueryRow(query, id, userID)
	} else {
		row = s.db.QueryRow(query, id)
	}
	r, err := scanInvoiceRequest(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: invoice request not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get invoice request: %w", err)
	}
	return r, nil
}

// GetInvoiceRequestOrders returns linked orders for a request.
func (s *PgStore) GetInvoiceRequestOrders(requestID int64) ([]InvoiceRequestOrder, error) {
	rows, err := s.db.Query(
		`SELECT id, request_id, order_id, amount FROM invoice_request_orders WHERE request_id=$1`, requestID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list request orders: %w", err)
	}
	defer rows.Close()

	var orders []InvoiceRequestOrder
	for rows.Next() {
		var o InvoiceRequestOrder
		if err := rows.Scan(&o.ID, &o.RequestID, &o.OrderID, &o.Amount); err != nil {
			return nil, fmt.Errorf("store: scan request order: %w", err)
		}
		orders = append(orders, o)
	}
	if orders == nil {
		orders = []InvoiceRequestOrder{}
	}
	return orders, nil
}

// CancelInvoiceRequest cancels a pending request.
func (s *PgStore) CancelInvoiceRequest(id int64, userID string) error {
	result, err := s.db.Exec(
		`UPDATE invoice_requests SET status='cancelled', updated_at=NOW() WHERE id=$1 AND user_id=$2 AND status='pending'`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("store: cancel invoice request: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: request not found or not in pending status")
	}
	return nil
}

// UpdateInvoiceRequestStatus updates request status (admin: pending -> processing).
func (s *PgStore) UpdateInvoiceRequestStatus(id int64, status string) error {
	result, err := s.db.Exec(
		`UPDATE invoice_requests SET status=$1, updated_at=NOW() WHERE id=$2 AND status='pending'`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("store: update invoice request status: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: invoice request not found")
	}
	return nil
}

// CompleteInvoiceRequest marks request as completed with file path and number.
// Allowed from 'processing' or 'completed' status (re-upload).
func (s *PgStore) CompleteInvoiceRequest(id int64, filePath, invoiceNumber string) error {
	result, err := s.db.Exec(
		`UPDATE invoice_requests SET status='completed', invoice_file_path=$1, invoice_number=$2, completed_at=NOW(), updated_at=NOW()
		 WHERE id=$3 AND status IN ('processing', 'completed')`,
		filePath, invoiceNumber, id,
	)
	if err != nil {
		return fmt.Errorf("store: complete invoice request: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: request not found or not in processing/completed status")
	}
	return nil
}

// RejectInvoiceRequest marks request as rejected with reason.
func (s *PgStore) RejectInvoiceRequest(id int64, reason string) error {
	result, err := s.db.Exec(
		`UPDATE invoice_requests SET status='rejected', reject_reason=$1, updated_at=NOW()
		 WHERE id=$2 AND status IN ('pending', 'processing')`,
		reason, id,
	)
	if err != nil {
		return fmt.Errorf("store: reject invoice request: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: request not found or already completed")
	}
	return nil
}

// AdminListInvoiceRequests returns all requests with pagination and optional status filter.
func (s *PgStore) AdminListInvoiceRequests(status string, limit, offset int) ([]InvoiceRequestDetail, int, error) {
	countQuery := `SELECT COUNT(*) FROM invoice_requests`
	listQuery := `SELECT ir.id, ir.user_id, ir.title_id, ir.invoice_type, ir.total_amount, ir.status, ir.remark, ir.reject_reason, ir.invoice_file_path, ir.invoice_number, ir.risk_level, ir.risk_reasons, ir.completed_at, ir.created_at, ir.updated_at,
		COALESCE(u.phone, u.email, NULLIF(u.nickname, ''), SUBSTRING(u.id::text, 1, 8))
		FROM invoice_requests ir JOIN users u ON u.id = ir.user_id`

	if status != "" {
		countQuery += ` WHERE status=$1`
		listQuery += ` WHERE ir.status=$1 ORDER BY ir.created_at DESC LIMIT $2 OFFSET $3`
	} else {
		listQuery += ` ORDER BY ir.created_at DESC LIMIT $1 OFFSET $2`
	}

	var total int
	if status != "" {
		err := s.db.QueryRow(countQuery, status).Scan(&total)
		if err != nil {
			return nil, 0, fmt.Errorf("store: count admin requests: %w", err)
		}
	} else {
		err := s.db.QueryRow(countQuery).Scan(&total)
		if err != nil {
			return nil, 0, fmt.Errorf("store: count admin requests: %w", err)
		}
	}

	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = s.db.Query(listQuery, status, limit, offset)
	} else {
		rows, err = s.db.Query(listQuery, limit, offset)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("store: list admin requests: %w", err)
	}
	defer rows.Close()

	var details []InvoiceRequestDetail
	for rows.Next() {
		var d InvoiceRequestDetail
		err := rows.Scan(&d.ID, &d.UserID, &d.TitleID, &d.InvoiceType, &d.TotalAmount,
			&d.Status, &d.Remark, &d.RejectReason, &d.InvoiceFilePath, &d.InvoiceNumber,
			&d.RiskLevel, &d.RiskReasons, &d.CompletedAt, &d.CreatedAt, &d.UpdatedAt, &d.UserIdentifier)
		if err != nil {
			return nil, 0, fmt.Errorf("store: scan admin request: %w", err)
		}
		details = append(details, d)
	}
	if details == nil {
		details = []InvoiceRequestDetail{}
	}
	return details, total, nil
}

// GetInvoiceTitleByID returns a title by ID without user filter (admin use).
func (s *PgStore) GetInvoiceTitleByID(id int64) (*InvoiceTitle, error) {
	row := s.db.QueryRow(
		`SELECT `+invoiceTitleColumns+` FROM invoice_titles WHERE id=$1`, id,
	)
	t, err := scanInvoiceTitle(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: invoice title not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get invoice title by id: %w", err)
	}
	return t, nil
}
