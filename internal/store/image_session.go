package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ImageSession represents an image generation session.
type ImageSession struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateImageSession creates a new image session.
func (s *PgStore) CreateImageSession(ctx context.Context, session *ImageSession) (*ImageSession, error) {
	query := `
		INSERT INTO image_sessions (user_id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query, session.UserID, session.Name).Scan(
		&session.ID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create image session: %w", err)
	}

	return session, nil
}

// GetImageSessions retrieves all sessions for a user, ordered by created_at DESC.
// Sessions linked to an image_share_key (i.e. used by external image-share key holders)
// are excluded from the owner's regular list view.
func (s *PgStore) GetImageSessions(ctx context.Context, userID string) ([]ImageSession, error) {
	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM image_sessions
		WHERE user_id = $1 AND image_share_key_id IS NULL
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query image sessions: %w", err)
	}
	defer rows.Close()

	var sessions []ImageSession
	for rows.Next() {
		var session ImageSession
		if err := rows.Scan(&session.ID, &session.UserID, &session.Name, &session.CreatedAt, &session.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan image session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return sessions, nil
}

// EnsureImageShareSession returns the dedicated image session for an image-share key,
// creating it on first use. The session belongs to the owner (so existing checks like
// session.UserID == userID continue to pass when userID is the owner) and is tagged with
// image_share_key_id so it is hidden from the owner's regular session list.
func (s *PgStore) EnsureImageShareSession(ctx context.Context, ownerUserID, imageShareKeyID, name string) (*ImageSession, error) {
	var session ImageSession
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, created_at, updated_at
		 FROM image_sessions
		 WHERE image_share_key_id = $1
		 LIMIT 1`,
		imageShareKeyID,
	).Scan(&session.ID, &session.UserID, &session.Name, &session.CreatedAt, &session.UpdatedAt)
	if err == nil {
		return &session, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to lookup image-share session: %w", err)
	}

	if name == "" {
		name = "image-share"
	}
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO image_sessions (user_id, name, image_share_key_id, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 RETURNING id, user_id, name, created_at, updated_at`,
		ownerUserID, name, imageShareKeyID,
	).Scan(&session.ID, &session.UserID, &session.Name, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create image-share session: %w", err)
	}
	return &session, nil
}

// GetImageSession retrieves a single session by ID.
func (s *PgStore) GetImageSession(ctx context.Context, sessionID int) (*ImageSession, error) {
	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM image_sessions
		WHERE id = $1
	`

	var session ImageSession
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.Name,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get image session: %w", err)
	}

	return &session, nil
}

// UpdateImageSession updates a session's name and updated_at timestamp.
func (s *PgStore) UpdateImageSession(ctx context.Context, sessionID int, name string) error {
	query := `
		UPDATE image_sessions
		SET name = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := s.db.ExecContext(ctx, query, name, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update image session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// DeleteImageSession deletes a session and all its generations (cascade).
func (s *PgStore) DeleteImageSession(ctx context.Context, sessionID int) error {
	query := `DELETE FROM image_sessions WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete image session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}
