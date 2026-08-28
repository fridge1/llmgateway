package store

import (
	"fmt"
	"time"
)

// AlertRule is a configurable threshold rule evaluated by the alerting service.
type AlertRule struct {
	ID              int       `json:"id"`
	Metric          string    `json:"metric"`
	DisplayName     string    `json:"display_name"`
	Threshold       int64     `json:"threshold"`
	CooldownSeconds int       `json:"cooldown_seconds"`
	Enabled         bool      `json:"enabled"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AlertEvent records a fired alert (also serves as the cooldown dedup record).
type AlertEvent struct {
	ID        int64     `json:"id"`
	Metric    string    `json:"metric"`
	Message   string    `json:"message"`
	Value     int64     `json:"value"`
	Threshold int64     `json:"threshold"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *PgStore) ListAlertRules() ([]AlertRule, error) {
	rows, err := s.db.Query(
		`SELECT id, metric, display_name, threshold, cooldown_seconds, enabled, updated_at
		 FROM alert_rules ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("store: list alert rules: %w", err)
	}
	defer rows.Close()
	var list []AlertRule
	for rows.Next() {
		var a AlertRule
		if err := rows.Scan(&a.ID, &a.Metric, &a.DisplayName, &a.Threshold, &a.CooldownSeconds, &a.Enabled, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan alert rule: %w", err)
		}
		list = append(list, a)
	}
	return list, nil
}

func (s *PgStore) UpdateAlertRule(id int, threshold int64, cooldownSeconds int, enabled bool) error {
	res, err := s.db.Exec(
		`UPDATE alert_rules SET threshold=$2, cooldown_seconds=$3, enabled=$4, updated_at=now() WHERE id=$1`,
		id, threshold, cooldownSeconds, enabled)
	if err != nil {
		return fmt.Errorf("store: update alert rule: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("alert rule not found")
	}
	return nil
}

// TryClaimAlertEvent atomically records an alert event unless another event for
// the same metric was recorded within the cooldown window. Returns true if the
// event was recorded (caller should notify), false if suppressed by cooldown.
func (s *PgStore) TryClaimAlertEvent(metric, message string, value, threshold int64, cooldownSeconds int) (bool, error) {
	res, err := s.db.Exec(
		`INSERT INTO alert_events (metric, message, value, threshold)
		 SELECT $1, $2, $3, $4
		 WHERE NOT EXISTS (
		   SELECT 1 FROM alert_events
		   WHERE metric = $1 AND created_at > now() - make_interval(secs => $5)
		 )`,
		metric, message, value, threshold, cooldownSeconds)
	if err != nil {
		return false, fmt.Errorf("store: claim alert event: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *PgStore) ListAlertEvents(limit, offset int) ([]AlertEvent, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM alert_events`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count alert events: %w", err)
	}
	rows, err := s.db.Query(
		`SELECT id, metric, message, value, threshold, created_at
		 FROM alert_events ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list alert events: %w", err)
	}
	defer rows.Close()
	var list []AlertEvent
	for rows.Next() {
		var e AlertEvent
		if err := rows.Scan(&e.ID, &e.Metric, &e.Message, &e.Value, &e.Threshold, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan alert event: %w", err)
		}
		list = append(list, e)
	}
	return list, total, nil
}

// ListAdminUsers returns all users with the admin role (alert recipients).
func (s *PgStore) ListAdminUsers() ([]User, error) {
	rows, err := s.db.Query(
		`SELECT id, phone, password_hash, COALESCE(nickname,''), role, status,
		        first_recharge_bonus_granted, created_at, updated_at
		 FROM users WHERE role = 'admin' AND status = 'active'`)
	if err != nil {
		return nil, fmt.Errorf("store: list admin users: %w", err)
	}
	defer rows.Close()
	var list []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Phone, &u.PasswordHash, &u.Nickname, &u.Role, &u.Status,
			&u.FirstRechargeBonusGranted, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan admin user: %w", err)
		}
		list = append(list, u)
	}
	return list, nil
}
