package store

import (
	"database/sql"
	"time"
)

// BlockedIP represents a blocked IP address entry.
type BlockedIP struct {
	IPAddress string     `json:"ip_address"`
	Reason    string     `json:"reason"`
	BlockedAt time.Time  `json:"blocked_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	BlockedBy *string    `json:"blocked_by"`
	Notes     *string    `json:"notes"`
}

// IsIPBlocked checks if an IP is currently blocked (not expired).
func (s *PgStore) IsIPBlocked(ipAddress string) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM blocked_ips
		WHERE ip_address = $1
		AND (expires_at IS NULL OR expires_at > NOW())
	`
	err := s.db.QueryRow(query, ipAddress).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// BlockIP adds an IP to the blocklist.
func (s *PgStore) BlockIP(ipAddress, reason, blockedBy string, expiresAt *time.Time) error {
	query := `
		INSERT INTO blocked_ips (ip_address, reason, blocked_at, expires_at, blocked_by, notes)
		VALUES ($1, $2, NOW(), $3, $4, NULL)
		ON CONFLICT (ip_address) DO UPDATE SET
			reason = EXCLUDED.reason,
			blocked_at = EXCLUDED.blocked_at,
			expires_at = EXCLUDED.expires_at,
			blocked_by = EXCLUDED.blocked_by
	`
	_, err := s.db.Exec(query, ipAddress, reason, expiresAt, blockedBy)
	return err
}

// UnblockIP removes an IP from the blocklist.
func (s *PgStore) UnblockIP(ipAddress string) error {
	query := `DELETE FROM blocked_ips WHERE ip_address = $1`
	_, err := s.db.Exec(query, ipAddress)
	return err
}

// GetBlockedIP retrieves a single blocked IP entry.
func (s *PgStore) GetBlockedIP(ipAddress string) (*BlockedIP, error) {
	var ip BlockedIP
	query := `
		SELECT ip_address, reason, blocked_at, expires_at, blocked_by, notes
		FROM blocked_ips
		WHERE ip_address = $1
	`
	err := s.db.QueryRow(query, ipAddress).Scan(
		&ip.IPAddress, &ip.Reason, &ip.BlockedAt, &ip.ExpiresAt, &ip.BlockedBy, &ip.Notes,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ip, nil
}

// ListBlockedIPs returns all blocked IPs with pagination.
func (s *PgStore) ListBlockedIPs(limit, offset int) ([]BlockedIP, error) {
	var ips []BlockedIP

	// Data query - only return active (non-expired) IPs
	query := `
		SELECT ip_address, reason, blocked_at, expires_at, blocked_by, notes
		FROM blocked_ips
		WHERE expires_at IS NULL OR expires_at > NOW()
		ORDER BY blocked_at DESC LIMIT $1 OFFSET $2
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ip BlockedIP
		if err := rows.Scan(&ip.IPAddress, &ip.Reason, &ip.BlockedAt, &ip.ExpiresAt, &ip.BlockedBy, &ip.Notes); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}

	return ips, rows.Err()
}

// CleanupExpiredBlockedIPs removes expired IP blocks and returns the count deleted.
func (s *PgStore) CleanupExpiredBlockedIPs() (int, error) {
	query := `DELETE FROM blocked_ips WHERE expires_at IS NOT NULL AND expires_at <= NOW()`
	result, err := s.db.Exec(query)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rowsAffected), nil
}

// CountBlockedIPs returns the current count of active blocked IPs.
func (s *PgStore) CountBlockedIPs() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM blocked_ips WHERE expires_at IS NULL OR expires_at > NOW()`
	err := s.db.QueryRow(query).Scan(&count)
	return count, err
}
