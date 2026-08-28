package imageshare

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Store handles persistence for image_share_keys.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store backed by the given *sql.DB.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ErrNotFound is returned when a key cannot be located.
var ErrNotFound = errors.New("imageshare: key not found")

// ErrQuotaExhausted is returned when a conditional update would exceed quota_total.
var ErrQuotaExhausted = errors.New("imageshare: quota exhausted")

// CreateKey inserts a new key row.
func (s *Store) CreateKey(ownerUserID, keyHash, keyPrefix, name string, quotaTotal int) (*Key, error) {
	if quotaTotal < 0 {
		return nil, fmt.Errorf("imageshare: quota_total must be >= 0")
	}
	var k Key
	err := s.db.QueryRow(
		`INSERT INTO image_share_keys (owner_user_id, key_hash, key_prefix, name, quota_total)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, owner_user_id, key_hash, key_prefix, name, quota_total, quota_used, status, last_used_at, created_at, updated_at`,
		ownerUserID, keyHash, keyPrefix, name, quotaTotal,
	).Scan(&k.ID, &k.OwnerUserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.QuotaTotal, &k.QuotaUsed, &k.Status, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("imageshare: create key: %w", err)
	}
	return &k, nil
}

// ListKeysByOwner returns all keys belonging to the given owner.
func (s *Store) ListKeysByOwner(ownerUserID string) ([]Key, error) {
	rows, err := s.db.Query(
		`SELECT id, owner_user_id, key_hash, key_prefix, name, quota_total, quota_used, status, last_used_at, created_at, updated_at
		 FROM image_share_keys WHERE owner_user_id = $1 ORDER BY created_at DESC`,
		ownerUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("imageshare: list keys: %w", err)
	}
	defer rows.Close()
	out := make([]Key, 0, 16)
	for rows.Next() {
		var k Key
		if err := rows.Scan(&k.ID, &k.OwnerUserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.QuotaTotal, &k.QuotaUsed, &k.Status, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, fmt.Errorf("imageshare: scan key: %w", err)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// GetKeyByHash looks up a key by its SHA-256 hash. Used for login.
func (s *Store) GetKeyByHash(keyHash string) (*Key, error) {
	var k Key
	err := s.db.QueryRow(
		`SELECT id, owner_user_id, key_hash, key_prefix, name, quota_total, quota_used, status, last_used_at, created_at, updated_at
		 FROM image_share_keys WHERE key_hash = $1`,
		keyHash,
	).Scan(&k.ID, &k.OwnerUserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.QuotaTotal, &k.QuotaUsed, &k.Status, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("imageshare: get key by hash: %w", err)
	}
	return &k, nil
}

// GetKey returns a key by id, scoped by owner.
func (s *Store) GetKey(id, ownerUserID string) (*Key, error) {
	var k Key
	err := s.db.QueryRow(
		`SELECT id, owner_user_id, key_hash, key_prefix, name, quota_total, quota_used, status, last_used_at, created_at, updated_at
		 FROM image_share_keys WHERE id = $1 AND owner_user_id = $2`,
		id, ownerUserID,
	).Scan(&k.ID, &k.OwnerUserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.QuotaTotal, &k.QuotaUsed, &k.Status, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("imageshare: get key: %w", err)
	}
	return &k, nil
}

// GetKeyByID looks up a key by its primary key (no owner scoping). Used by middleware.
func (s *Store) GetKeyByID(id string) (*Key, error) {
	var k Key
	err := s.db.QueryRow(
		`SELECT id, owner_user_id, key_hash, key_prefix, name, quota_total, quota_used, status, last_used_at, created_at, updated_at
		 FROM image_share_keys WHERE id = $1`,
		id,
	).Scan(&k.ID, &k.OwnerUserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.QuotaTotal, &k.QuotaUsed, &k.Status, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("imageshare: get key by id: %w", err)
	}
	return &k, nil
}

// UpdatePatch describes optional fields to update on a key.
type UpdatePatch struct {
	Name       *string
	Status     *string
	QuotaTotal *int
	ResetUsed  bool
}

// UpdateKey applies the given patch to a key, scoped by owner.
func (s *Store) UpdateKey(id, ownerUserID string, p UpdatePatch) (*Key, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("imageshare: begin tx: %w", err)
	}
	defer tx.Rollback()

	if p.Name != nil {
		if _, err := tx.Exec(`UPDATE image_share_keys SET name = $1, updated_at = NOW() WHERE id = $2 AND owner_user_id = $3`, *p.Name, id, ownerUserID); err != nil {
			return nil, fmt.Errorf("imageshare: update name: %w", err)
		}
	}
	if p.Status != nil {
		if *p.Status != "active" && *p.Status != "disabled" {
			return nil, fmt.Errorf("imageshare: invalid status: %s", *p.Status)
		}
		if _, err := tx.Exec(`UPDATE image_share_keys SET status = $1, updated_at = NOW() WHERE id = $2 AND owner_user_id = $3`, *p.Status, id, ownerUserID); err != nil {
			return nil, fmt.Errorf("imageshare: update status: %w", err)
		}
	}
	if p.QuotaTotal != nil {
		if *p.QuotaTotal < 0 {
			return nil, fmt.Errorf("imageshare: quota_total must be >= 0")
		}
		if _, err := tx.Exec(`UPDATE image_share_keys SET quota_total = $1, updated_at = NOW() WHERE id = $2 AND owner_user_id = $3`, *p.QuotaTotal, id, ownerUserID); err != nil {
			return nil, fmt.Errorf("imageshare: update quota_total: %w", err)
		}
	}
	if p.ResetUsed {
		if _, err := tx.Exec(`UPDATE image_share_keys SET quota_used = 0, updated_at = NOW() WHERE id = $1 AND owner_user_id = $2`, id, ownerUserID); err != nil {
			return nil, fmt.Errorf("imageshare: reset used: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("imageshare: commit: %w", err)
	}
	return s.GetKey(id, ownerUserID)
}

// DeleteKey removes a key, scoped by owner.
func (s *Store) DeleteKey(id, ownerUserID string) error {
	res, err := s.db.Exec(`DELETE FROM image_share_keys WHERE id = $1 AND owner_user_id = $2`, id, ownerUserID)
	if err != nil {
		return fmt.Errorf("imageshare: delete key: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// IncrementUsed atomically increments quota_used by n, only when status='active'
// and the increment would not exceed quota_total. Returns the new quota_used on success
// or ErrQuotaExhausted when the conditional update affects zero rows.
func (s *Store) IncrementUsed(id string, n int) (int, error) {
	if n <= 0 {
		return 0, nil
	}
	var newUsed int
	err := s.db.QueryRow(
		`UPDATE image_share_keys
		 SET quota_used = quota_used + $1, last_used_at = NOW(), updated_at = NOW()
		 WHERE id = $2 AND status = 'active' AND quota_used + $1 <= quota_total
		 RETURNING quota_used`,
		n, id,
	).Scan(&newUsed)
	if err == sql.ErrNoRows {
		return 0, ErrQuotaExhausted
	}
	if err != nil {
		return 0, fmt.Errorf("imageshare: increment used: %w", err)
	}
	return newUsed, nil
}

// RefundUsed atomically decrements quota_used by n. Used to roll back a
// reservation made at submission time when the corresponding task fully fails
// or could not be enqueued. Caps at 0 defensively in case of double refund.
func (s *Store) RefundUsed(id string, n int) error {
	if n <= 0 {
		return nil
	}
	_, err := s.db.Exec(
		`UPDATE image_share_keys
		 SET quota_used = GREATEST(quota_used - $1, 0), updated_at = NOW()
		 WHERE id = $2`,
		n, id,
	)
	if err != nil {
		return fmt.Errorf("imageshare: refund used: %w", err)
	}
	return nil
}

// TouchLastUsed updates last_used_at to now.
func (s *Store) TouchLastUsed(id string) error {
	_, err := s.db.Exec(`UPDATE image_share_keys SET last_used_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("imageshare: touch last used: %w", err)
	}
	return nil
}

// IsImageShareEnabled returns whether the given user has the image-share feature enabled
// and is currently active.
func (s *Store) IsImageShareEnabled(userID string) (enabled bool, status string, err error) {
	err = s.db.QueryRow(
		`SELECT image_share_enabled, status FROM users WHERE id = $1`,
		userID,
	).Scan(&enabled, &status)
	if err == sql.ErrNoRows {
		return false, "", ErrNotFound
	}
	if err != nil {
		return false, "", fmt.Errorf("imageshare: check enabled: %w", err)
	}
	return enabled, status, nil
}

// SetImageShareEnabled toggles users.image_share_enabled.
func (s *Store) SetImageShareEnabled(userID string, enabled bool) error {
	res, err := s.db.Exec(
		`UPDATE users SET image_share_enabled = $1, updated_at = NOW() WHERE id = $2`,
		enabled, userID,
	)
	if err != nil {
		return fmt.Errorf("imageshare: set enabled: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// touch updates updated_at — used to bump activity timestamps in tests.
func (s *Store) touch(id string) error {
	_, err := s.db.Exec(`UPDATE image_share_keys SET updated_at = NOW() WHERE id = $1`, id)
	return err
}

// since lints unused imports in case some helpers go unused early; remove if not needed.
var _ = time.Now
