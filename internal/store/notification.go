package store

import "time"

// Notification represents a user notification (system message, etc.).
type Notification struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	IsRead    bool      `json:"is_read"`
	RefType   *string   `json:"ref_type"`
	RefID     *string   `json:"ref_id"`
	CreatedAt time.Time `json:"created_at"`
}
