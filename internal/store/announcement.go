package store

import (
	"database/sql"
	"fmt"
	"time"
)

// Announcement represents a system announcement.
type Announcement struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DisplayMode string     `json:"display_mode"`
	CreatedBy   *string    `json:"created_by,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (s *PgStore) ListAnnouncements(limit, offset int) ([]Announcement, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM announcements`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count announcements: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT id, title, content, status, priority, display_mode, created_by, published_at, created_at, updated_at
		 FROM announcements ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list announcements: %w", err)
	}
	defer rows.Close()

	var list []Announcement
	for rows.Next() {
		var a Announcement
		if err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.Status, &a.Priority, &a.DisplayMode, &a.CreatedBy, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan announcement: %w", err)
		}
		list = append(list, a)
	}
	return list, total, nil
}

func (s *PgStore) CreateAnnouncement(title, content, status, priority, displayMode, createdBy string) (*Announcement, error) {
	var a Announcement
	var publishedAt *time.Time
	if status == "published" {
		now := time.Now()
		publishedAt = &now
	}
	err := s.db.QueryRow(
		`INSERT INTO announcements (title, content, status, priority, display_mode, created_by, published_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, title, content, status, priority, display_mode, created_by, published_at, created_at, updated_at`,
		title, content, status, priority, displayMode, createdBy, publishedAt,
	).Scan(&a.ID, &a.Title, &a.Content, &a.Status, &a.Priority, &a.DisplayMode, &a.CreatedBy, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create announcement: %w", err)
	}
	return &a, nil
}

func (s *PgStore) GetAnnouncementByID(id int64) (*Announcement, error) {
	var a Announcement
	err := s.db.QueryRow(
		`SELECT id, title, content, status, priority, display_mode, created_by, published_at, created_at, updated_at
		 FROM announcements WHERE id = $1`, id,
	).Scan(&a.ID, &a.Title, &a.Content, &a.Status, &a.Priority, &a.DisplayMode, &a.CreatedBy, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("announcement not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get announcement: %w", err)
	}
	return &a, nil
}

func (s *PgStore) UpdateAnnouncement(id int64, title, content, status, priority, displayMode string) (*Announcement, error) {
	// If status is changing to published, set published_at
	var publishedAtExpr string
	if status == "published" {
		publishedAtExpr = `CASE WHEN published_at IS NULL THEN NOW() ELSE published_at END`
	} else {
		publishedAtExpr = `published_at`
	}

	var a Announcement
	query := fmt.Sprintf(
		`UPDATE announcements SET title=$1, content=$2, status=$3, priority=$4, display_mode=$5, published_at=%s, updated_at=NOW()
		 WHERE id=$6
		 RETURNING id, title, content, status, priority, display_mode, created_by, published_at, created_at, updated_at`,
		publishedAtExpr,
	)
	err := s.db.QueryRow(query, title, content, status, priority, displayMode, id).
		Scan(&a.ID, &a.Title, &a.Content, &a.Status, &a.Priority, &a.DisplayMode, &a.CreatedBy, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("announcement not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: update announcement: %w", err)
	}
	return &a, nil
}

func (s *PgStore) DeleteAnnouncement(id int64) error {
	res, err := s.db.Exec(`DELETE FROM announcements WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete announcement: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("announcement not found")
	}
	return nil
}

func (s *PgStore) ListPublishedAnnouncements() ([]Announcement, error) {
	rows, err := s.db.Query(
		`SELECT id, title, content, status, priority, display_mode, created_by, published_at, created_at, updated_at
		 FROM announcements WHERE status = 'published' ORDER BY published_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list published announcements: %w", err)
	}
	defer rows.Close()

	var list []Announcement
	for rows.Next() {
		var a Announcement
		if err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.Status, &a.Priority, &a.DisplayMode, &a.CreatedBy, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan announcement: %w", err)
		}
		list = append(list, a)
	}
	return list, nil
}
