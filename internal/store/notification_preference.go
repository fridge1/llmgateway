package store

import (
	"fmt"
	"time"
)

// NotificationPreference is a per-user channel switch for one event type.
type NotificationPreference struct {
	UserID    string    `json:"user_id"`
	EventType string    `json:"event_type"`
	Channel   string    `json:"channel"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NotificationEventTypes enumerates event types users may configure.
var NotificationEventTypes = []string{
	"balance_low",
	"subscription_expiry",
	"ticket",
	"ops_alert",
	"lottery_win",          // 活动抽奖中奖
	"lottery_lose",         // 活动抽奖未中奖
	"recharge_lottery_win", // 充值抽奖中奖
}

// ValidNotificationEventType reports whether t is configurable.
func ValidNotificationEventType(t string) bool {
	for _, e := range NotificationEventTypes {
		if e == t {
			return true
		}
	}
	return false
}

// ListNotificationPreferences returns all stored preference rows for a user.
// Absent rows mean "default" (SMS off).
func (s *PgStore) ListNotificationPreferences(userID string) ([]NotificationPreference, error) {
	rows, err := s.db.Query(
		`SELECT user_id, event_type, channel, enabled, updated_at
		 FROM notification_preferences WHERE user_id=$1`, userID)
	if err != nil {
		return nil, fmt.Errorf("store: list notification preferences: %w", err)
	}
	defer rows.Close()
	var list []NotificationPreference
	for rows.Next() {
		var p NotificationPreference
		if err := rows.Scan(&p.UserID, &p.EventType, &p.Channel, &p.Enabled, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan notification preference: %w", err)
		}
		list = append(list, p)
	}
	return list, nil
}

// UpsertNotificationPreference sets one (event, channel) switch for a user.
func (s *PgStore) UpsertNotificationPreference(userID, eventType, channel string, enabled bool) error {
	_, err := s.db.Exec(
		`INSERT INTO notification_preferences (user_id, event_type, channel, enabled)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (user_id, event_type, channel)
		 DO UPDATE SET enabled=EXCLUDED.enabled, updated_at=now()`,
		userID, eventType, channel, enabled)
	if err != nil {
		return fmt.Errorf("store: upsert notification preference: %w", err)
	}
	return nil
}

// SMSNotificationEnabled reports whether the user opted into SMS for eventType.
// Missing row = false (opt-in model).
func (s *PgStore) SMSNotificationEnabled(userID, eventType string) (bool, error) {
	var enabled bool
	err := s.db.QueryRow(
		`SELECT enabled FROM notification_preferences
		 WHERE user_id=$1 AND event_type=$2 AND channel='sms'`,
		userID, eventType).Scan(&enabled)
	if err != nil {
		return false, nil // no row = default off; treat scan errors as off too
	}
	return enabled, nil
}

// ListExpiringSubscriptions returns user IDs whose active subscription expires
// within the given number of days (for the expiry-reminder job).
func (s *PgStore) ListExpiringSubscriptions(withinDays int) ([]UserSubscription, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, plan_id, status, expires_at
		 FROM user_subscriptions
		 WHERE status = 'active'
		   AND expires_at > now()
		   AND expires_at < now() + make_interval(days => $1)`,
		withinDays)
	if err != nil {
		return nil, fmt.Errorf("store: list expiring subscriptions: %w", err)
	}
	defer rows.Close()
	var list []UserSubscription
	for rows.Next() {
		var us UserSubscription
		if err := rows.Scan(&us.ID, &us.UserID, &us.PlanID, &us.Status, &us.ExpiresAt); err != nil {
			return nil, fmt.Errorf("store: scan expiring subscription: %w", err)
		}
		list = append(list, us)
	}
	return list, nil
}
