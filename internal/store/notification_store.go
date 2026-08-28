package store

import (
	"fmt"
	"strings"
)

const notifCols = `id, user_id, type, title, content, is_read, ref_type, ref_id, created_at`

func scanNotif(row interface{ Scan(...any) error }, n *Notification) error {
	return row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Content, &n.IsRead, &n.RefType, &n.RefID, &n.CreatedAt)
}

func (s *PgStore) CreateNotification(userID, nType, title, content string, refType, refID *string) (*Notification, error) {
	var n Notification
	err := s.db.QueryRow(
		`INSERT INTO notifications (user_id, type, title, content, ref_type, ref_id)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING `+notifCols,
		userID, nType, title, content, refType, refID,
	).Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Content, &n.IsRead, &n.RefType, &n.RefID, &n.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create notification: %w", err)
	}
	return &n, nil
}

func (s *PgStore) BatchCreateNotifications(notifications []Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	const batchSize = 500
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: batch create notifications begin: %w", err)
	}
	defer tx.Rollback()

	for i := 0; i < len(notifications); i += batchSize {
		end := i + batchSize
		if end > len(notifications) {
			end = len(notifications)
		}
		batch := notifications[i:end]

		var sb strings.Builder
		sb.WriteString(`INSERT INTO notifications (user_id, type, title, content, ref_type, ref_id) VALUES `)
		args := make([]any, 0, len(batch)*6)
		for j, n := range batch {
			if j > 0 {
				sb.WriteString(",")
			}
			base := j * 6
			sb.WriteString(fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d)", base+1, base+2, base+3, base+4, base+5, base+6))
			args = append(args, n.UserID, n.Type, n.Title, n.Content, n.RefType, n.RefID)
		}
		if _, err := tx.Exec(sb.String(), args...); err != nil {
			return fmt.Errorf("store: batch create notifications exec: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: batch create notifications commit: %w", err)
	}
	return nil
}

// TryClaimNotificationDedup atomically records that a (user, kind) notification
// was sent today. Returns true if this is the first claim today (caller should
// send the notification), false if already sent (caller should skip).
func (s *PgStore) TryClaimNotificationDedup(userID, kind string) (bool, error) {
	res, err := s.db.Exec(
		`INSERT INTO notification_dedup (user_id, kind, sent_on)
		 VALUES ($1, $2, CURRENT_DATE)
		 ON CONFLICT (user_id, kind, sent_on) DO NOTHING`,
		userID, kind,
	)
	if err != nil {
		return false, fmt.Errorf("store: claim notification dedup: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *PgStore) ListNotifications(userID string, limit, offset int) ([]Notification, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id=$1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count notifications: %w", err)
	}
	rows, err := s.db.Query(
		`SELECT `+notifCols+` FROM notifications WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list notifications: %w", err)
	}
	defer rows.Close()
	var list []Notification
	for rows.Next() {
		var n Notification
		if err := scanNotif(rows, &n); err != nil {
			return nil, 0, fmt.Errorf("store: scan notification: %w", err)
		}
		list = append(list, n)
	}
	return list, total, nil
}

func (s *PgStore) CountUnreadNotifications(userID string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND is_read=FALSE`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: count unread notifications: %w", err)
	}
	return count, nil
}

func (s *PgStore) MarkNotificationRead(userID string, id int64) error {
	res, err := s.db.Exec(`UPDATE notifications SET is_read=TRUE WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return fmt.Errorf("store: mark notification read: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

func (s *PgStore) MarkAllNotificationsRead(userID string) error {
	_, err := s.db.Exec(`UPDATE notifications SET is_read=TRUE WHERE user_id=$1 AND is_read=FALSE`, userID)
	if err != nil {
		return fmt.Errorf("store: mark all notifications read: %w", err)
	}
	return nil
}
