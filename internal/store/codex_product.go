package store

import (
	"database/sql"
	"fmt"
	"time"
)

type CodexProduct struct {
	ID          int       `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceCNY    float64   `json:"price_cny"`
	SortOrder   int       `json:"sort_order"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListCodexProducts 列出所有启用的 Codex 商品
func (s *PgStore) ListCodexProducts() ([]CodexProduct, error) {
	rows, err := s.db.Query(`
		SELECT id, sku, name, description, price_cny, sort_order, status, created_at, updated_at
		FROM codex_products
		WHERE status = 'active'
		ORDER BY sort_order ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list codex products: %w", err)
	}
	defer rows.Close()

	var products []CodexProduct
	for rows.Next() {
		var p CodexProduct
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceCNY,
			&p.SortOrder, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan codex product: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate codex products: %w", err)
	}
	return products, nil
}

// GetCodexProductByID 根据 ID 获取商品
func (s *PgStore) GetCodexProductByID(id int) (*CodexProduct, error) {
	var p CodexProduct
	err := s.db.QueryRow(`
		SELECT id, sku, name, description, price_cny, sort_order, status, created_at, updated_at
		FROM codex_products WHERE id = $1
	`, id).Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceCNY,
		&p.SortOrder, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: codex product not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get codex product: %w", err)
	}
	return &p, nil
}

// ListAllCodexProducts 列出所有商品（管理后台用）
func (s *PgStore) ListAllCodexProducts() ([]CodexProduct, error) {
	rows, err := s.db.Query(`
		SELECT id, sku, name, description, price_cny, sort_order, status, created_at, updated_at
		FROM codex_products
		ORDER BY sort_order ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list all codex products: %w", err)
	}
	defer rows.Close()

	var products []CodexProduct
	for rows.Next() {
		var p CodexProduct
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceCNY,
			&p.SortOrder, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan codex product: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate all codex products: %w", err)
	}
	return products, nil
}

// CreateCodexProduct creates a new Codex product (admin).
func (s *PgStore) CreateCodexProduct(sku, name, description string, priceCNY float64, sortOrder int, status string) (*CodexProduct, error) {
	if status == "" {
		status = "active"
	}
	var p CodexProduct
	err := s.db.QueryRow(`
		INSERT INTO codex_products (sku, name, description, price_cny, sort_order, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, sku, name, description, price_cny, sort_order, status, created_at, updated_at
	`, sku, name, description, priceCNY, sortOrder, status).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceCNY, &p.SortOrder, &p.Status, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: create codex product: %w", err)
	}
	return &p, nil
}

// UpdateCodexProduct updates an existing Codex product (admin).
func (s *PgStore) UpdateCodexProduct(id int, sku, name, description string, priceCNY float64, sortOrder int, status string) (*CodexProduct, error) {
	if status == "" {
		status = "active"
	}
	var p CodexProduct
	err := s.db.QueryRow(`
		UPDATE codex_products
		SET sku = $1, name = $2, description = $3, price_cny = $4, sort_order = $5, status = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING id, sku, name, description, price_cny, sort_order, status, created_at, updated_at
	`, sku, name, description, priceCNY, sortOrder, status, id).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceCNY, &p.SortOrder, &p.Status, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: codex product not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: update codex product: %w", err)
	}
	return &p, nil
}

// DeleteCodexProduct deletes a Codex product by id (admin). Refuses if the
// product still has non-terminal orders to avoid orphaning order history.
func (s *PgStore) DeleteCodexProduct(id int) error {
	// 检查是否仍有 pending/paid 订单引用该商品，避免删除后订单失去商品关联。
	var inUse int
	if err := s.db.QueryRow(`
		SELECT COUNT(*) FROM codex_orders
		WHERE product_id = $1 AND status IN ('pending', 'paid')
	`, id).Scan(&inUse); err != nil {
		return fmt.Errorf("store: check codex product in use: %w", err)
	}
	if inUse > 0 {
		return fmt.Errorf("store: cannot delete codex product %d: %d active order(s) reference it", id, inUse)
	}
	res, err := s.db.Exec(`DELETE FROM codex_products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete codex product: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: codex product not found")
	}
	return nil
}
