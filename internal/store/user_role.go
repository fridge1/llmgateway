package store

import "fmt"

// backofficeRoles enumerates assignable platform roles.
var backofficeRoles = map[string]bool{"user": true, "admin": true, "support": true, "finance": true, "ops": true}

// ValidPlatformRole reports whether role is an assignable platform role.
func ValidPlatformRole(role string) bool { return backofficeRoles[role] }

// UpdateUserRole changes a user's platform role.
func (s *PgStore) UpdateUserRole(id, role string) error {
	if !ValidPlatformRole(role) {
		return fmt.Errorf("invalid role %q", role)
	}
	res, err := s.db.Exec(
		"UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2",
		role, id,
	)
	if err != nil {
		return fmt.Errorf("store: update user role: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("store: user not found")
	}
	return nil
}
