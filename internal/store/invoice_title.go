package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// InvoiceTitle represents a user's invoice title (抬头).
type InvoiceTitle struct {
	ID          int64     `json:"id"`
	UserID      string    `json:"user_id"`
	Type        string    `json:"type"`        // "personal" | "enterprise"
	TitleName   string    `json:"title_name"`
	TaxNumber   string    `json:"tax_number"`
	BankName    string    `json:"bank_name"`
	BankAccount string    `json:"bank_account"`
	Address     string    `json:"address"`
	Phone       string    `json:"phone"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const invoiceTitleColumns = `id, user_id, type, title_name, tax_number, bank_name, bank_account, address, phone, is_default, created_at, updated_at`

func scanInvoiceTitle(row interface{ Scan(...any) error }) (*InvoiceTitle, error) {
	var t InvoiceTitle
	err := row.Scan(&t.ID, &t.UserID, &t.Type, &t.TitleName, &t.TaxNumber,
		&t.BankName, &t.BankAccount, &t.Address, &t.Phone, &t.IsDefault,
		&t.CreatedAt, &t.UpdatedAt)
	return &t, err
}

func (s *PgStore) CreateInvoiceTitle(userID, titleType, titleName, taxNumber, bankName, bankAccount, address, phone string) (*InvoiceTitle, error) {
	row := s.db.QueryRow(
		`INSERT INTO invoice_titles (user_id, type, title_name, tax_number, bank_name, bank_account, address, phone)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+invoiceTitleColumns,
		userID, titleType, titleName, taxNumber, bankName, bankAccount, address, phone,
	)
	t, err := scanInvoiceTitle(row)
	if err != nil {
		return nil, fmt.Errorf("store: create invoice title: %w", err)
	}
	return t, nil
}

func (s *PgStore) UpdateInvoiceTitle(id int64, userID, titleType, titleName, taxNumber, bankName, bankAccount, address, phone string) (*InvoiceTitle, error) {
	row := s.db.QueryRow(
		`UPDATE invoice_titles SET type=$1, title_name=$2, tax_number=$3, bank_name=$4, bank_account=$5, address=$6, phone=$7, updated_at=NOW()
		 WHERE id=$8 AND user_id=$9
		 RETURNING `+invoiceTitleColumns,
		titleType, titleName, taxNumber, bankName, bankAccount, address, phone, id, userID,
	)
	t, err := scanInvoiceTitle(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: invoice title not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: update invoice title: %w", err)
	}
	return t, nil
}

func (s *PgStore) DeleteInvoiceTitle(id int64, userID string) error {
	result, err := s.db.Exec(`DELETE FROM invoice_titles WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		// PostgreSQL 外键约束错误包含 "foreign key constraint" 或 "violates"
		if strings.Contains(err.Error(), "foreign key") || strings.Contains(err.Error(), "violates") {
			return fmt.Errorf("store: invoice title is still referenced by requests")
		}
		return fmt.Errorf("store: delete invoice title: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: invoice title not found")
	}
	return nil
}

func (s *PgStore) ListInvoiceTitlesByUser(userID string) ([]InvoiceTitle, error) {
	rows, err := s.db.Query(
		`SELECT `+invoiceTitleColumns+` FROM invoice_titles WHERE user_id=$1 ORDER BY is_default DESC, created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list invoice titles: %w", err)
	}
	defer rows.Close()

	var titles []InvoiceTitle
	for rows.Next() {
		t, err := scanInvoiceTitle(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan invoice title: %w", err)
		}
		titles = append(titles, *t)
	}
	if titles == nil {
		titles = []InvoiceTitle{}
	}
	return titles, nil
}

func (s *PgStore) GetInvoiceTitle(id int64, userID string) (*InvoiceTitle, error) {
	row := s.db.QueryRow(
		`SELECT `+invoiceTitleColumns+` FROM invoice_titles WHERE id=$1 AND user_id=$2`, id, userID,
	)
	t, err := scanInvoiceTitle(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: invoice title not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get invoice title: %w", err)
	}
	return t, nil
}

func (s *PgStore) SetDefaultInvoiceTitle(id int64, userID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Clear existing default
	if _, err := tx.Exec(`UPDATE invoice_titles SET is_default=false, updated_at=NOW() WHERE user_id=$1 AND is_default=true`, userID); err != nil {
		return fmt.Errorf("store: clear default: %w", err)
	}
	// Set new default
	result, err := tx.Exec(`UPDATE invoice_titles SET is_default=true, updated_at=NOW() WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return fmt.Errorf("store: set default: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: invoice title not found")
	}
	return tx.Commit()
}
