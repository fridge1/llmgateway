package store

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

// hashPassword hashes a password using bcrypt.
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ---------------------------------------------------------------------------
// Tenant CRUD
// ---------------------------------------------------------------------------

func (s *PgStore) CreateTenant(name, ownerID string) (*Tenant, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	var t Tenant
	err = tx.QueryRow(
		`INSERT INTO tenants (name, owner_id) VALUES ($1, $2)
		 RETURNING id, name, owner_id, status, created_at, updated_at`,
		name, ownerID,
	).Scan(&t.ID, &t.Name, &t.OwnerID, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create tenant: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO tenant_members (tenant_id, user_id, role) VALUES ($1, $2, 'owner')`,
		t.ID, ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: create tenant member: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO tenant_balances (tenant_id) VALUES ($1)`,
		t.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: create tenant balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit: %w", err)
	}
	return &t, nil
}

func (s *PgStore) GetTenantByID(id string) (*Tenant, error) {
	var t Tenant
	err := s.db.QueryRow(
		`SELECT id, name, owner_id, status, is_enterprise, created_by_admin,
		        COALESCE(contact_phone, '') as contact_phone,
		        COALESCE(contact_email, '') as contact_email,
		        created_at, updated_at
		 FROM tenants WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Name, &t.OwnerID, &t.Status, &t.IsEnterprise, &t.CreatedByAdmin, &t.ContactPhone, &t.ContactEmail, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get tenant: %w", err)
	}
	return &t, nil
}

func (s *PgStore) UpdateTenant(id, name string) error {
	_, err := s.db.Exec(
		`UPDATE tenants SET name = $1, updated_at = NOW() WHERE id = $2`,
		name, id,
	)
	if err != nil {
		return fmt.Errorf("store: update tenant: %w", err)
	}
	return nil
}

func (s *PgStore) ListTenantsByUser(userID string) ([]TenantWithRole, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.name, tm.role
		 FROM tenants t
		 JOIN tenant_members tm ON t.id = tm.tenant_id
		 WHERE tm.user_id = $1 AND t.status = 'active'
		 ORDER BY tm.joined_at`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list tenants by user: %w", err)
	}
	defer rows.Close()

	var result []TenantWithRole
	for rows.Next() {
		var tw TenantWithRole
		if err := rows.Scan(&tw.ID, &tw.Name, &tw.Role); err != nil {
			return nil, fmt.Errorf("store: scan tenant: %w", err)
		}
		result = append(result, tw)
	}
	return result, rows.Err()
}

func (s *PgStore) ListAllTenants(limit, offset int) ([]Tenant, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM tenants`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count tenants: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT id, name, owner_id, status, is_enterprise, created_by_admin,
		        COALESCE(contact_phone, '') as contact_phone,
		        COALESCE(contact_email, '') as contact_email,
		        created_at, updated_at
		 FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list all tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.OwnerID, &t.Status, &t.IsEnterprise, &t.CreatedByAdmin, &t.ContactPhone, &t.ContactEmail, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	return tenants, total, rows.Err()
}

func (s *PgStore) DeleteTenant(id string) error {
	_, err := s.db.Exec(`DELETE FROM tenants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete tenant: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tenant Members
// ---------------------------------------------------------------------------

func (s *PgStore) AddTenantMember(tenantID, userID, role string) error {
	_, err := s.db.Exec(
		`INSERT INTO tenant_members (tenant_id, user_id, role) VALUES ($1, $2, $3)`,
		tenantID, userID, role,
	)
	if err != nil {
		return fmt.Errorf("store: add tenant member: %w", err)
	}
	return nil
}

func (s *PgStore) RemoveTenantMember(tenantID, userID string) error {
	res, err := s.db.Exec(
		`DELETE FROM tenant_members WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	)
	if err != nil {
		return fmt.Errorf("store: remove tenant member: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: member not found")
	}
	return nil
}

func (s *PgStore) UpdateTenantMemberRole(tenantID, userID, role string) error {
	res, err := s.db.Exec(
		`UPDATE tenant_members SET role = $1 WHERE tenant_id = $2 AND user_id = $3`,
		role, tenantID, userID,
	)
	if err != nil {
		return fmt.Errorf("store: update tenant member role: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: member not found")
	}
	return nil
}

func (s *PgStore) ListTenantMembers(tenantID string) ([]TenantMember, error) {
	rows, err := s.db.Query(
		`SELECT tm.id, tm.tenant_id, tm.user_id, tm.role, tm.joined_at,
		        u.phone, COALESCE(u.nickname, '')
		 FROM tenant_members tm
		 JOIN users u ON tm.user_id = u.id
		 WHERE tm.tenant_id = $1
		 ORDER BY tm.joined_at`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list tenant members: %w", err)
	}
	defer rows.Close()

	var members []TenantMember
	for rows.Next() {
		var m TenantMember
		if err := rows.Scan(&m.ID, &m.TenantID, &m.UserID, &m.Role, &m.JoinedAt, &m.Phone, &m.Nickname); err != nil {
			return nil, fmt.Errorf("store: scan tenant member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *PgStore) GetTenantMember(tenantID, userID string) (*TenantMember, error) {
	var m TenantMember
	err := s.db.QueryRow(
		`SELECT id, tenant_id, user_id, role, joined_at
		 FROM tenant_members WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	).Scan(&m.ID, &m.TenantID, &m.UserID, &m.Role, &m.JoinedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: tenant member not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get tenant member: %w", err)
	}
	return &m, nil
}

func (s *PgStore) TransferTenantOwnership(tenantID, newOwnerID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE tenant_members SET role = 'admin' WHERE tenant_id = $1 AND role = 'owner'`,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: demote old owner: %w", err)
	}

	res, err := tx.Exec(
		`UPDATE tenant_members SET role = 'owner' WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, newOwnerID,
	)
	if err != nil {
		return fmt.Errorf("store: promote new owner: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: new owner is not a member")
	}

	_, err = tx.Exec(
		`UPDATE tenants SET owner_id = $1, updated_at = NOW() WHERE id = $2`,
		newOwnerID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: update tenant owner: %w", err)
	}

	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Tenant Balance
// ---------------------------------------------------------------------------

func (s *PgStore) GetTenantBalance(tenantID string) (*TenantBalance, error) {
	var b TenantBalance
	err := s.db.QueryRow(
		`SELECT tenant_id, balance, frozen, total_recharged, total_consumed
		 FROM tenant_balances WHERE tenant_id = $1`,
		tenantID,
	).Scan(&b.TenantID, &b.Balance, &b.Frozen, &b.TotalRecharged, &b.TotalConsumed)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: tenant balance not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get tenant balance: %w", err)
	}
	return &b, nil
}

func (s *PgStore) RechargeTenant(tenantID string, amount float64, operatorID, description string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Get tenant owner
	var ownerID string
	err = tx.QueryRow(`SELECT owner_id FROM tenants WHERE id = $1`, tenantID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("store: get tenant owner: %w", err)
	}

	// Recharge owner's personal balance
	res, err := tx.Exec(
		`UPDATE balances SET balance = balance + $1, updated_at = NOW() WHERE user_id = $2`,
		amount, ownerID,
	)
	if err != nil {
		return fmt.Errorf("store: recharge owner balance: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		_, err = tx.Exec(
			`INSERT INTO balances (user_id, balance, frozen) VALUES ($1, $2, 0)`,
			ownerID, amount,
		)
		if err != nil {
			return fmt.Errorf("store: insert owner balance: %w", err)
		}
	}

	// Update tenant_balances for statistics
	_, err = tx.Exec(
		`UPDATE tenant_balances SET balance = balance + $1, total_recharged = total_recharged + $1, updated_at = NOW()
		 WHERE tenant_id = $2`,
		amount, tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: recharge tenant: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO tenant_transactions (tenant_id, type, amount, operator_id, description)
		 VALUES ($1, 'recharge', $2, $3, $4)`,
		tenantID, amount, operatorID, description,
	)
	if err != nil {
		return fmt.Errorf("store: recharge tenant insert tx: %w", err)
	}

	return tx.Commit()
}

func (s *PgStore) FreezeTenantBalance(tenantID string, amount float64) error {
	res, err := s.db.Exec(
		`UPDATE tenant_balances SET frozen = frozen + $1, updated_at = NOW()
		 WHERE tenant_id = $2 AND (balance - frozen) >= $1`,
		amount, tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: freeze tenant balance: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: insufficient tenant balance")
	}
	return nil
}

func (s *PgStore) UnfreezeTenantBalance(tenantID string, amount float64) error {
	res, err := s.db.Exec(
		`UPDATE tenant_balances SET frozen = frozen - $1, updated_at = NOW()
		 WHERE tenant_id = $2 AND frozen >= $1`,
		amount, tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: unfreeze tenant balance: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: insufficient tenant frozen amount")
	}
	return nil
}

func (s *PgStore) SettleTenantBilling(tenantID string, frozenAmount, actualCost float64, model, requestID string, tokens TokenUsage, apiKeyID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE tenant_balances SET frozen = frozen - $1, updated_at = NOW() WHERE tenant_id = $2`,
		frozenAmount, tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: settle tenant unfreeze: %w", err)
	}

	_, err = tx.Exec(
		`UPDATE tenant_balances SET balance = balance - $1, total_consumed = total_consumed + $1, updated_at = NOW() WHERE tenant_id = $2`,
		actualCost, tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: settle tenant balance: %w", err)
	}

	var apiKeyIDPtr *string
	if apiKeyID != "" {
		apiKeyIDPtr = &apiKeyID
	}

	_, err = tx.Exec(
		`INSERT INTO tenant_transactions (tenant_id, type, amount, model, request_id, api_key_id,
		  prompt_tokens, completion_tokens, cache_read_tokens, cache_creation_tokens, description)
		 VALUES ($1, 'consumption', $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		tenantID, actualCost, model, requestID, apiKeyIDPtr,
		intPtrOrNil(tokens.PromptTokens), intPtrOrNil(tokens.CompletionTokens),
		intPtrOrNil(tokens.CacheReadTokens), intPtrOrNil(tokens.CacheCreationTokens),
		fmt.Sprintf("API call to %s", model),
	)
	if err != nil {
		return fmt.Errorf("store: settle tenant insert tx: %w", err)
	}

	return tx.Commit()
}

func (s *PgStore) DirectChargeTenant(tenantID string, amount float64, model, requestID string, tokens TokenUsage, apiKeyID string) error {
	if amount <= 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE tenant_balances SET balance = balance - $1, total_consumed = total_consumed + $1, updated_at = NOW()
		 WHERE tenant_id = $2`,
		amount, tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: direct charge tenant: %w", err)
	}

	var apiKeyIDPtr *string
	if apiKeyID != "" {
		apiKeyIDPtr = &apiKeyID
	}

	_, err = tx.Exec(
		`INSERT INTO tenant_transactions (tenant_id, type, amount, model, request_id, api_key_id,
		  prompt_tokens, completion_tokens, cache_read_tokens, cache_creation_tokens, description)
		 VALUES ($1, 'consumption', $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		tenantID, amount, model, requestID, apiKeyIDPtr,
		intPtrOrNil(tokens.PromptTokens), intPtrOrNil(tokens.CompletionTokens),
		intPtrOrNil(tokens.CacheReadTokens), intPtrOrNil(tokens.CacheCreationTokens),
		fmt.Sprintf("API call to %s", model),
	)
	if err != nil {
		return fmt.Errorf("store: direct charge tenant insert tx: %w", err)
	}

	return tx.Commit()
}

func (s *PgStore) ListTenantTransactions(tenantID string, limit, offset int, typeFilter string, startDate, endDate *time.Time) ([]Transaction, int, error) {
	query := `SELECT id, type, amount, model, request_id, description, created_at,
	                 prompt_tokens, completion_tokens, cache_read_tokens, cache_creation_tokens
	          FROM tenant_transactions WHERE tenant_id = $1`
	countQuery := `SELECT COUNT(*) FROM tenant_transactions WHERE tenant_id = $1`
	args := []any{tenantID}
	countArgs := []any{tenantID}
	idx := 2

	if typeFilter != "" {
		query += fmt.Sprintf(` AND type = $%d`, idx)
		countQuery += fmt.Sprintf(` AND type = $%d`, idx)
		args = append(args, typeFilter)
		countArgs = append(countArgs, typeFilter)
		idx++
	}
	if startDate != nil {
		query += fmt.Sprintf(` AND created_at >= $%d`, idx)
		countQuery += fmt.Sprintf(` AND created_at >= $%d`, idx)
		args = append(args, *startDate)
		countArgs = append(countArgs, *startDate)
		idx++
	}
	if endDate != nil {
		query += fmt.Sprintf(` AND created_at <= $%d`, idx)
		countQuery += fmt.Sprintf(` AND created_at <= $%d`, idx)
		args = append(args, *endDate)
		countArgs = append(countArgs, *endDate)
		idx++
	}

	var total int
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count tenant transactions: %w", err)
	}

	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, idx, idx+1)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list tenant transactions: %w", err)
	}
	defer rows.Close()

	var txns []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Type, &t.Amount, &t.Model, &t.RequestID, &t.Description, &t.CreatedAt,
			&t.PromptTokens, &t.CompletionTokens, &t.CacheReadTokens, &t.CacheCreationTokens); err != nil {
			return nil, 0, fmt.Errorf("store: scan tenant transaction: %w", err)
		}
		txns = append(txns, t)
	}
	return txns, total, rows.Err()
}

// ---------------------------------------------------------------------------
// Tenant API Keys
// ---------------------------------------------------------------------------

func (s *PgStore) CreateTenantAPIKey(tenantID, keyHash, keyPrefix, name, createdBy string) (*TenantAPIKey, error) {
	var k TenantAPIKey
	err := s.db.QueryRow(
		`INSERT INTO tenant_api_keys (tenant_id, key_hash, key_prefix, name, created_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, tenant_id, name, key_prefix, created_by, status, last_used_at, created_at`,
		tenantID, keyHash, keyPrefix, name, createdBy,
	).Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyPrefix, &k.CreatedBy, &k.Status, &k.LastUsedAt, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create tenant api key: %w", err)
	}
	return &k, nil
}

func (s *PgStore) ListTenantAPIKeys(tenantID string) ([]TenantAPIKey, error) {
	rows, err := s.db.Query(
		`SELECT id, tenant_id, name, key_prefix, created_by, status, last_used_at, created_at
		 FROM tenant_api_keys WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list tenant api keys: %w", err)
	}
	defer rows.Close()

	var keys []TenantAPIKey
	for rows.Next() {
		var k TenantAPIKey
		if err := rows.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyPrefix, &k.CreatedBy, &k.Status, &k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan tenant api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *PgStore) GetTenantAPIKeyByHash(keyHash string) (*TenantAPIKey, error) {
	var k TenantAPIKey
	err := s.db.QueryRow(
		`SELECT id, tenant_id, name, key_hash, key_prefix, created_by, status, last_used_at, created_at
		 FROM tenant_api_keys WHERE key_hash = $1 AND status = 'active'`,
		keyHash,
	).Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.CreatedBy, &k.Status, &k.LastUsedAt, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: tenant api key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get tenant api key by hash: %w", err)
	}
	return &k, nil
}

func (s *PgStore) DeleteTenantAPIKey(id, tenantID string) error {
	res, err := s.db.Exec(
		`DELETE FROM tenant_api_keys WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: delete tenant api key: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: tenant api key not found")
	}
	return nil
}

func (s *PgStore) TouchTenantAPIKeyLastUsed(id string) error {
	_, err := s.db.Exec(
		`UPDATE tenant_api_keys SET last_used_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

// ---------------------------------------------------------------------------
// Tenant Invitations
// ---------------------------------------------------------------------------

func (s *PgStore) CreateTenantInvitation(tenantID, phone, role, invitedBy string) (*TenantInvitation, error) {
	var inv TenantInvitation
	err := s.db.QueryRow(
		`INSERT INTO tenant_invitations (tenant_id, phone, role, invited_by, expires_at)
		 VALUES ($1, $2, $3, $4, NOW() + INTERVAL '7 days')
		 ON CONFLICT (tenant_id, phone) DO UPDATE SET role = $3, invited_by = $4, status = 'pending', expires_at = NOW() + INTERVAL '7 days'
		 RETURNING id, tenant_id, phone, role, invited_by, status, expires_at, created_at`,
		tenantID, phone, role, invitedBy,
	).Scan(&inv.ID, &inv.TenantID, &inv.Phone, &inv.Role, &inv.InvitedBy, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create tenant invitation: %w", err)
	}
	return &inv, nil
}

func (s *PgStore) ListTenantInvitations(tenantID string) ([]TenantInvitation, error) {
	rows, err := s.db.Query(
		`SELECT id, tenant_id, phone, role, invited_by, status, expires_at, created_at
		 FROM tenant_invitations WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list tenant invitations: %w", err)
	}
	defer rows.Close()

	var invitations []TenantInvitation
	for rows.Next() {
		var inv TenantInvitation
		if err := rows.Scan(&inv.ID, &inv.TenantID, &inv.Phone, &inv.Role, &inv.InvitedBy, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan tenant invitation: %w", err)
		}
		invitations = append(invitations, inv)
	}
	return invitations, rows.Err()
}

func (s *PgStore) AcceptTenantInvitation(invitationID, userID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	var tenantID, role string
	err = tx.QueryRow(
		`UPDATE tenant_invitations SET status = 'accepted'
		 WHERE id = $1 AND status = 'pending' AND expires_at > NOW()
		 RETURNING tenant_id, role`,
		invitationID,
	).Scan(&tenantID, &role)
	if err == sql.ErrNoRows {
		return fmt.Errorf("store: invitation not found or expired")
	}
	if err != nil {
		return fmt.Errorf("store: accept invitation: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO tenant_members (tenant_id, user_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT (tenant_id, user_id) DO NOTHING`,
		tenantID, userID, role,
	)
	if err != nil {
		return fmt.Errorf("store: add member from invitation: %w", err)
	}

	return tx.Commit()
}

func (s *PgStore) DeleteTenantInvitation(id string) error {
	_, err := s.db.Exec(`DELETE FROM tenant_invitations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete tenant invitation: %w", err)
	}
	return nil
}

func (s *PgStore) GetPendingInvitationsByPhone(phone string) ([]TenantInvitation, error) {
	rows, err := s.db.Query(
		`SELECT ti.id, ti.tenant_id, ti.phone, ti.role, ti.invited_by, ti.status, ti.expires_at, ti.created_at,
		        t.name
		 FROM tenant_invitations ti
		 JOIN tenants t ON ti.tenant_id = t.id
		 WHERE ti.phone = $1 AND ti.status = 'pending' AND ti.expires_at > NOW()
		 ORDER BY ti.created_at DESC`,
		phone,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get pending invitations: %w", err)
	}
	defer rows.Close()

	var invitations []TenantInvitation
	for rows.Next() {
		var inv TenantInvitation
		if err := rows.Scan(&inv.ID, &inv.TenantID, &inv.Phone, &inv.Role, &inv.InvitedBy, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &inv.TenantName); err != nil {
			return nil, fmt.Errorf("store: scan pending invitation: %w", err)
		}
		invitations = append(invitations, inv)
	}
	return invitations, rows.Err()
}

// ---------------------------------------------------------------------------
// Enterprise Tenant Methods
// ---------------------------------------------------------------------------

func (s *PgStore) CreateEnterpriseTenant(name, ownerID, adminID, contactPhone, contactEmail string) (*Tenant, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	var t Tenant
	err = tx.QueryRow(
		`INSERT INTO tenants (name, owner_id, is_enterprise, created_by_admin, contact_phone, contact_email)
		 VALUES ($1, $2, true, $3, $4, $5)
		 RETURNING id, name, owner_id, status, is_enterprise, created_by_admin,
		           COALESCE(contact_phone, '') as contact_phone,
		           COALESCE(contact_email, '') as contact_email,
		           created_at, updated_at`,
		name, ownerID, adminID, contactPhone, contactEmail,
	).Scan(&t.ID, &t.Name, &t.OwnerID, &t.Status, &t.IsEnterprise, &t.CreatedByAdmin, &t.ContactPhone, &t.ContactEmail, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create enterprise tenant: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO tenant_members (tenant_id, user_id, role) VALUES ($1, $2, 'owner')`,
		t.ID, ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: create tenant member: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO tenant_balances (tenant_id) VALUES ($1)`,
		t.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: create tenant balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit: %w", err)
	}
	return &t, nil
}

func (s *PgStore) UpdateTenantEnterpriseInfo(tenantID, contactPhone, contactEmail string) error {
	_, err := s.db.Exec(
		`UPDATE tenants SET contact_phone = $1, contact_email = $2, updated_at = NOW() WHERE id = $3`,
		contactPhone, contactEmail, tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: update tenant enterprise info: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tenant Sub-User Methods
// ---------------------------------------------------------------------------

func (s *PgStore) CreateTenantSubUser(tenantID, username, password, nickname string, quotaLimit *float64, createdBy string) (*TenantSubUser, error) {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("store: hash password: %w", err)
	}

	var su TenantSubUser
	err = s.db.QueryRow(
		`INSERT INTO tenant_sub_users (tenant_id, username, password_hash, nickname, quota_limit, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, tenant_id, username, nickname, status, quota_limit, quota_used, created_by, created_at, updated_at`,
		tenantID, username, passwordHash, nickname, quotaLimit, createdBy,
	).Scan(&su.ID, &su.TenantID, &su.Username, &su.Nickname, &su.Status, &su.QuotaLimit, &su.QuotaUsed, &su.CreatedBy, &su.CreatedAt, &su.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create tenant sub-user: %w", err)
	}
	return &su, nil
}

func (s *PgStore) GetTenantSubUser(subUserID string) (*TenantSubUser, error) {
	var su TenantSubUser
	err := s.db.QueryRow(
		`SELECT id, tenant_id, username, nickname, status, quota_limit, quota_used, created_by, created_at, updated_at
		 FROM tenant_sub_users WHERE id = $1`,
		subUserID,
	).Scan(&su.ID, &su.TenantID, &su.Username, &su.Nickname, &su.Status, &su.QuotaLimit, &su.QuotaUsed, &su.CreatedBy, &su.CreatedAt, &su.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: sub-user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get tenant sub-user: %w", err)
	}
	return &su, nil
}

func (s *PgStore) GetTenantSubUserByUsername(tenantID, username string) (*TenantSubUser, error) {
	var su TenantSubUser
	err := s.db.QueryRow(
		`SELECT id, tenant_id, username, nickname, status, quota_limit, quota_used, created_by, created_at, updated_at
		 FROM tenant_sub_users WHERE tenant_id = $1 AND username = $2`,
		tenantID, username,
	).Scan(&su.ID, &su.TenantID, &su.Username, &su.Nickname, &su.Status, &su.QuotaLimit, &su.QuotaUsed, &su.CreatedBy, &su.CreatedAt, &su.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: sub-user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get tenant sub-user by username: %w", err)
	}
	return &su, nil
}

func (s *PgStore) AuthenticateSubUser(tenantID, username, password string) (*TenantSubUser, error) {
	var su TenantSubUser
	var passwordHash string
	err := s.db.QueryRow(
		`SELECT id, tenant_id, username, password_hash, nickname, status, quota_limit, quota_used, created_by, created_at, updated_at
		 FROM tenant_sub_users WHERE tenant_id = $1 AND username = $2`,
		tenantID, username,
	).Scan(&su.ID, &su.TenantID, &su.Username, &passwordHash, &su.Nickname, &su.Status, &su.QuotaLimit, &su.QuotaUsed, &su.CreatedBy, &su.CreatedAt, &su.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: invalid credentials")
	}
	if err != nil {
		return nil, fmt.Errorf("store: authenticate sub-user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("store: invalid credentials")
	}

	return &su, nil
}

func (s *PgStore) ListTenantSubUsers(tenantID string) ([]TenantSubUserWithQuota, error) {
	rows, err := s.db.Query(
		`SELECT id, tenant_id, username, nickname, status, quota_limit, quota_used, created_by, created_at, updated_at
		 FROM tenant_sub_users WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list tenant sub-users: %w", err)
	}
	defer rows.Close()

	var result []TenantSubUserWithQuota
	for rows.Next() {
		var su TenantSubUserWithQuota
		if err := rows.Scan(&su.ID, &su.TenantID, &su.Username, &su.Nickname, &su.Status, &su.QuotaLimit, &su.QuotaUsed, &su.CreatedBy, &su.CreatedAt, &su.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan sub-user: %w", err)
		}
		if su.QuotaLimit != nil {
			remaining := *su.QuotaLimit - su.QuotaUsed
			su.QuotaRemaining = &remaining
		}
		result = append(result, su)
	}
	return result, rows.Err()
}

func (s *PgStore) UpdateTenantSubUserQuota(subUserID string, quotaLimit *float64) error {
	_, err := s.db.Exec(
		`UPDATE tenant_sub_users SET quota_limit = $1, updated_at = NOW() WHERE id = $2`,
		quotaLimit, subUserID,
	)
	if err != nil {
		return fmt.Errorf("store: update sub-user quota: %w", err)
	}
	return nil
}

func (s *PgStore) UpdateTenantSubUserStatus(subUserID, status string) error {
	_, err := s.db.Exec(
		`UPDATE tenant_sub_users SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, subUserID,
	)
	if err != nil {
		return fmt.Errorf("store: update sub-user status: %w", err)
	}
	return nil
}

func (s *PgStore) DeleteTenantSubUser(subUserID string) error {
	_, err := s.db.Exec(`DELETE FROM tenant_sub_users WHERE id = $1`, subUserID)
	if err != nil {
		return fmt.Errorf("store: delete sub-user: %w", err)
	}
	return nil
}

func (s *PgStore) ResetTenantSubUserPassword(subUserID, newPassword string) error {
	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("store: hash password: %w", err)
	}

	_, err = s.db.Exec(
		`UPDATE tenant_sub_users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		passwordHash, subUserID,
	)
	if err != nil {
		return fmt.Errorf("store: reset sub-user password: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tenant Sub-User Quota Methods
// ---------------------------------------------------------------------------

func (s *PgStore) IncrementSubUserQuotaUsed(subUserID string, amount float64) error {
	_, err := s.db.Exec(
		`UPDATE tenant_sub_users SET quota_used = quota_used + $1, updated_at = NOW() WHERE id = $2`,
		amount, subUserID,
	)
	if err != nil {
		return fmt.Errorf("store: increment sub-user quota used: %w", err)
	}
	return nil
}

func (s *PgStore) CheckSubUserQuota(subUserID string, estimatedCost float64) error {
	var quotaLimit sql.NullFloat64
	var quotaUsed float64

	err := s.db.QueryRow(
		`SELECT quota_limit, quota_used FROM tenant_sub_users WHERE id = $1`,
		subUserID,
	).Scan(&quotaLimit, &quotaUsed)
	if err != nil {
		return fmt.Errorf("store: check sub-user quota: %w", err)
	}

	if !quotaLimit.Valid {
		return nil
	}

	remaining := quotaLimit.Float64 - quotaUsed
	if remaining < estimatedCost {
		return fmt.Errorf("insufficient quota: remaining %.4f, required %.4f", remaining, estimatedCost)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tenant Sub-User API Key Methods
// ---------------------------------------------------------------------------

func (s *PgStore) CreateTenantSubUserKey(subUserID, tenantID, keyHash, keyPrefix, name string) (*TenantSubUserKey, error) {
	var key TenantSubUserKey
	err := s.db.QueryRow(
		`INSERT INTO tenant_sub_user_keys (sub_user_id, tenant_id, key_hash, key_prefix, name)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, sub_user_id, tenant_id, name, key_prefix, status, last_used_at, created_at`,
		subUserID, tenantID, keyHash, keyPrefix, name,
	).Scan(&key.ID, &key.SubUserID, &key.TenantID, &key.Name, &key.KeyPrefix, &key.Status, &key.LastUsedAt, &key.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create sub-user key: %w", err)
	}
	key.KeyHash = keyHash
	return &key, nil
}

func (s *PgStore) ListTenantSubUserKeys(subUserID string) ([]TenantSubUserKey, error) {
	rows, err := s.db.Query(
		`SELECT id, sub_user_id, tenant_id, name, key_prefix, status, last_used_at, created_at
		 FROM tenant_sub_user_keys WHERE sub_user_id = $1 ORDER BY created_at DESC`,
		subUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list sub-user keys: %w", err)
	}
	defer rows.Close()

	var keys []TenantSubUserKey
	for rows.Next() {
		var key TenantSubUserKey
		if err := rows.Scan(&key.ID, &key.SubUserID, &key.TenantID, &key.Name, &key.KeyPrefix, &key.Status, &key.LastUsedAt, &key.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan sub-user key: %w", err)
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (s *PgStore) GetTenantSubUserKeyByHash(keyHash string) (*TenantSubUserKey, error) {
	var key TenantSubUserKey
	err := s.db.QueryRow(
		`SELECT id, sub_user_id, tenant_id, name, key_hash, key_prefix, status, last_used_at, created_at
		 FROM tenant_sub_user_keys WHERE key_hash = $1 AND status = 'active'`,
		keyHash,
	).Scan(&key.ID, &key.SubUserID, &key.TenantID, &key.Name, &key.KeyHash, &key.KeyPrefix, &key.Status, &key.LastUsedAt, &key.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: sub-user key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get sub-user key by hash: %w", err)
	}
	return &key, nil
}

func (s *PgStore) DeleteTenantSubUserKey(keyID, tenantID string) error {
	res, err := s.db.Exec(`DELETE FROM tenant_sub_user_keys WHERE id = $1 AND tenant_id = $2`, keyID, tenantID)
	if err != nil {
		return fmt.Errorf("store: delete sub-user key: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: delete sub-user key rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("store: sub-user key not found")
	}
	return nil
}

func (s *PgStore) DeleteSubUserOwnKey(keyID, subUserID string) error {
	res, err := s.db.Exec(`DELETE FROM tenant_sub_user_keys WHERE id = $1 AND sub_user_id = $2`, keyID, subUserID)
	if err != nil {
		return fmt.Errorf("store: delete sub-user own key: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: delete sub-user own key rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("store: sub-user key not found")
	}
	return nil
}

func (s *PgStore) TouchTenantSubUserKeyLastUsed(keyID string) error {
	_, err := s.db.Exec(
		`UPDATE tenant_sub_user_keys SET last_used_at = NOW() WHERE id = $1`,
		keyID,
	)
	if err != nil {
		return fmt.Errorf("store: touch sub-user key last used: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tenant Sub-User Transaction Methods
// ---------------------------------------------------------------------------

func (s *PgStore) ListTenantSubUserTransactions(subUserID string, limit, offset int) ([]Transaction, int, error) {
	rows, err := s.db.Query(
		`SELECT id, tenant_id AS user_id, type, amount, 0 AS balance_after, model, request_id, description, created_at
		 FROM tenant_transactions
		 WHERE sub_user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		subUserID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list sub-user transactions: %w", err)
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.BalanceAfter, &t.Model, &t.RequestID, &t.Description, &t.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}

	var total int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM tenant_transactions WHERE sub_user_id = $1`, subUserID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("store: count sub-user transactions: %w", err)
	}

	return transactions, total, rows.Err()
}

func (s *PgStore) ListTenantAllSubUserTransactions(tenantID string, limit, offset int, subUserID string) ([]Transaction, int, error) {
	query := `SELECT id, tenant_id AS user_id, type, amount, 0 AS balance_after, model, request_id, description, created_at,
		       prompt_tokens, completion_tokens, cache_read_tokens, cache_creation_tokens,
		       sub_user_id, sub_user_username
		 FROM tenant_transactions
		 WHERE tenant_id = $1`
	countQuery := `SELECT COUNT(*) FROM tenant_transactions WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	countArgs := []interface{}{tenantID}

	if subUserID != "" {
		query += ` AND sub_user_id = $2`
		countQuery += ` AND sub_user_id = $2`
		args = append(args, subUserID)
		countArgs = append(countArgs, subUserID)
	}

	query += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list tenant all sub-user transactions: %w", err)
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.BalanceAfter, &t.Model, &t.RequestID, &t.Description, &t.CreatedAt,
			&t.PromptTokens, &t.CompletionTokens, &t.CacheReadTokens, &t.CacheCreationTokens,
			&t.SubUserID, &t.SubUserUsername); err != nil {
			return nil, 0, fmt.Errorf("store: scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}

	var total int
	err = s.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("store: count tenant all sub-user transactions: %w", err)
	}

	return transactions, total, rows.Err()
}

func (s *PgStore) RecordSubUserTransaction(tenantID, subUserID string, amount float64, model, requestID string, tokens TokenUsage, apiKeyID string) error {
	var username string
	err := s.db.QueryRow(`SELECT username FROM tenant_sub_users WHERE id = $1`, subUserID).Scan(&username)
	if err != nil {
		return fmt.Errorf("store: get sub-user username: %w", err)
	}

	var apiKeyIDPtr *string
	if apiKeyID != "" {
		apiKeyIDPtr = &apiKeyID
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE tenant_balances SET total_consumed = total_consumed + $1, updated_at = NOW()
		 WHERE tenant_id = $2`,
		amount, tenantID,
	)
	if err != nil {
		return fmt.Errorf("store: update tenant total_consumed: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO tenant_transactions (tenant_id, sub_user_id, sub_user_username, type, amount, model, request_id, api_key_id,
		  prompt_tokens, completion_tokens, cache_read_tokens, cache_creation_tokens, description)
		 VALUES ($1, $2, $3, 'consumption', $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		tenantID, subUserID, username, amount, model, requestID, apiKeyIDPtr,
		intPtrOrNil(tokens.PromptTokens), intPtrOrNil(tokens.CompletionTokens),
		intPtrOrNil(tokens.CacheReadTokens), intPtrOrNil(tokens.CacheCreationTokens),
		fmt.Sprintf("子用户 %s 调用 %s", username, model),
	)
	if err != nil {
		return fmt.Errorf("store: record sub-user transaction: %w", err)
	}

	return tx.Commit()
}

// GetTenantBillingStats returns comprehensive billing statistics for a tenant.
func (s *PgStore) GetTenantBillingStats(tenantID string, days int) (*TenantBillingStats, error) {
	stats := &TenantBillingStats{}

	err := s.db.QueryRow(
		`SELECT
			COALESCE(SUM(CASE WHEN created_at >= CURRENT_DATE THEN amount END), 0),
			COALESCE(SUM(amount), 0)
		 FROM tenant_transactions
		 WHERE tenant_id = $1 AND type = 'consumption'
		 AND created_at >= date_trunc('month', CURRENT_DATE)`,
		tenantID,
	).Scan(&stats.TodayCost, &stats.MonthCost)
	if err != nil {
		return nil, fmt.Errorf("store: tenant billing stats costs: %w", err)
	}

	g := new(errgroup.Group)

	g.Go(func() error {
		rows, err := s.db.Query(
			`SELECT d::date AS date, COALESCE(SUM(t.amount), 0) AS cost
			 FROM generate_series(CURRENT_DATE - $2::int * INTERVAL '1 day', CURRENT_DATE, '1 day') d
			 LEFT JOIN tenant_transactions t ON t.tenant_id = $1 AND t.type = 'consumption'
			   AND t.created_at::date = d::date
			 GROUP BY d::date ORDER BY d::date`,
			tenantID, days,
		)
		if err != nil {
			return fmt.Errorf("store: tenant daily trend: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var dc DailyCost
			var d time.Time
			if err := rows.Scan(&d, &dc.Cost); err != nil {
				return fmt.Errorf("store: scan tenant daily cost: %w", err)
			}
			dc.Date = d.Format("2006-01-02")
			stats.DailyTrend = append(stats.DailyTrend, dc)
		}
		if stats.DailyTrend == nil {
			stats.DailyTrend = []DailyCost{}
		}
		return nil
	})

	g.Go(func() error {
		rows, err := s.db.Query(
			`SELECT model, SUM(amount) FROM tenant_transactions
			 WHERE tenant_id = $1 AND type = 'consumption'
			 AND model IS NOT NULL
			 AND created_at >= CURRENT_DATE - $2::int * INTERVAL '1 day'
			 GROUP BY model ORDER BY SUM(amount) DESC LIMIT 15`,
			tenantID, days,
		)
		if err != nil {
			return fmt.Errorf("store: tenant model breakdown: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var mc ModelCost
			if err := rows.Scan(&mc.Model, &mc.Cost); err != nil {
				return fmt.Errorf("store: scan tenant model cost: %w", err)
			}
			stats.ModelBreakdown = append(stats.ModelBreakdown, mc)
		}
		if stats.ModelBreakdown == nil {
			stats.ModelBreakdown = []ModelCost{}
		}
		return nil
	})

	g.Go(func() error {
		rows, err := s.db.Query(
			`SELECT sub_user_id, sub_user_username, SUM(amount), COUNT(*)
			 FROM tenant_transactions
			 WHERE tenant_id = $1 AND type = 'consumption'
			 AND created_at >= CURRENT_DATE - $2::int * INTERVAL '1 day'
			 AND sub_user_id IS NOT NULL
			 GROUP BY sub_user_id, sub_user_username
			 ORDER BY SUM(amount) DESC LIMIT 10`,
			tenantID, days,
		)
		if err != nil {
			return fmt.Errorf("store: tenant sub-user ranking: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var sc SubUserCost
			if err := rows.Scan(&sc.SubUserID, &sc.SubUserUsername, &sc.TotalCost, &sc.RequestCount); err != nil {
				return fmt.Errorf("store: scan sub-user cost: %w", err)
			}
			stats.SubUserRanking = append(stats.SubUserRanking, sc)
		}
		if stats.SubUserRanking == nil {
			stats.SubUserRanking = []SubUserCost{}
		}
		return nil
	})

	g.Go(func() error {
		return s.db.QueryRow(
			`SELECT
				COALESCE(SUM(prompt_tokens), 0),
				COALESCE(SUM(completion_tokens), 0),
				COALESCE(SUM(cache_read_tokens), 0),
				COALESCE(SUM(cache_creation_tokens), 0)
			 FROM tenant_transactions
			 WHERE tenant_id = $1 AND type = 'consumption'
			 AND created_at >= CURRENT_DATE - $2::int * INTERVAL '1 day'`,
			tenantID, days,
		).Scan(&stats.TokenStats.TotalPrompt, &stats.TokenStats.TotalCompletion,
			&stats.TokenStats.TotalCacheRead, &stats.TokenStats.TotalCacheCreation)
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if len(stats.DailyTrend) > 1 {
		var totalCost float64
		for _, dc := range stats.DailyTrend {
			totalCost += dc.Cost
		}
		stats.DailyAverage = totalCost / float64(len(stats.DailyTrend))
	}

	return stats, nil
}

// ListTenantTransactionsForExport returns all tenant transactions within a date range for export.
func (s *PgStore) ListTenantTransactionsForExport(tenantID string, startDate, endDate *time.Time, subUserID string) ([]Transaction, error) {
	query := `SELECT id, tenant_id AS user_id, type, amount, 0 AS balance_after, model, request_id, description, created_at,
		       prompt_tokens, completion_tokens, cache_read_tokens, cache_creation_tokens,
		       sub_user_id, sub_user_username
		 FROM tenant_transactions
		 WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	argIdx := 2

	if startDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *startDate)
		argIdx++
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *endDate)
		argIdx++
	}
	if subUserID != "" {
		query += fmt.Sprintf(" AND sub_user_id = $%d", argIdx)
		args = append(args, subUserID)
		argIdx++
	}

	query += " ORDER BY created_at DESC LIMIT 10000"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list tenant transactions for export: %w", err)
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.BalanceAfter, &t.Model, &t.RequestID, &t.Description, &t.CreatedAt,
			&t.PromptTokens, &t.CompletionTokens, &t.CacheReadTokens, &t.CacheCreationTokens,
			&t.SubUserID, &t.SubUserUsername); err != nil {
			return nil, fmt.Errorf("store: scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}

	return transactions, rows.Err()
}

// GetSubUserModelStats returns comprehensive model-level statistics for a sub-user.
func (s *PgStore) GetSubUserModelStats(subUserID string, startDate, endDate *time.Time) (*SubUserModelStats, error) {
	stats := &SubUserModelStats{}

	// Get sub-user info
	err := s.db.QueryRow(
		`SELECT id, username FROM tenant_sub_users WHERE id = $1`,
		subUserID,
	).Scan(&stats.SubUserID, &stats.SubUserUsername)
	if err != nil {
		return nil, fmt.Errorf("store: get sub-user info: %w", err)
	}

	// Build WHERE clause with date filters
	whereClause := "WHERE sub_user_id = $1 AND type = 'consumption'"
	args := []interface{}{subUserID}
	argIdx := 2

	if startDate != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *startDate)
		argIdx++
	}
	if endDate != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *endDate)
		argIdx++
	}

	// Set period info
	if startDate != nil || endDate != nil {
		stats.Period = &StatsPeriod{}
		if startDate != nil {
			stats.Period.StartDate = startDate.Format("2006-01-02")
		}
		if endDate != nil {
			stats.Period.EndDate = endDate.Format("2006-01-02")
		}
	}

	// Get total cost and request count
	err = s.db.QueryRow(
		fmt.Sprintf(`SELECT COALESCE(SUM(amount), 0), COUNT(*)
		 FROM tenant_transactions
		 %s`, whereClause),
		args...,
	).Scan(&stats.TotalCost, &stats.TotalRequests)
	if err != nil {
		return nil, fmt.Errorf("store: get sub-user total stats: %w", err)
	}

	// Get model breakdown
	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT
			model,
			SUM(amount) AS cost,
			COUNT(*) AS request_count,
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) AS cache_creation_tokens
		 FROM tenant_transactions
		 %s AND model IS NOT NULL
		 GROUP BY model
		 ORDER BY SUM(amount) DESC`, whereClause),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get sub-user model breakdown: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var mc SubUserModelCost
		if err := rows.Scan(&mc.Model, &mc.Cost, &mc.RequestCount, &mc.PromptTokens, &mc.CompletionTokens, &mc.CacheReadTokens, &mc.CacheCreationTokens); err != nil {
			return nil, fmt.Errorf("store: scan model cost: %w", err)
		}
		stats.ModelBreakdown = append(stats.ModelBreakdown, mc)
	}
	if stats.ModelBreakdown == nil {
		stats.ModelBreakdown = []SubUserModelCost{}
	}

	return stats, rows.Err()
}


