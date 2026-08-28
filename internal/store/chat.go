package store

import (
	"database/sql"
	"fmt"
	"time"
)

// ChatSession represents a chat session belonging to a user.
type ChatSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     *string   `json:"title,omitempty"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatMessage represents a single message within a chat session.
type ChatMessage struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	TokensUsed *int      `json:"tokens_used,omitempty"`
	Cost       *float64  `json:"cost,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ListSessions returns all chat sessions for a user, ordered by updated_at DESC.
func (s *PgStore) ListSessions(userID string) ([]ChatSession, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, title, model, created_at, updated_at
		 FROM chat_sessions WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []ChatSession
	for rows.Next() {
		var cs ChatSession
		if err := rows.Scan(&cs.ID, &cs.UserID, &cs.Title, &cs.Model, &cs.CreatedAt, &cs.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan session: %w", err)
		}
		sessions = append(sessions, cs)
	}
	if sessions == nil {
		sessions = []ChatSession{}
	}
	return sessions, nil
}

// CreateSession creates a new chat session for the given user.
func (s *PgStore) CreateSession(userID, model, title string) (*ChatSession, error) {
	var cs ChatSession
	var titleArg *string
	if title != "" {
		titleArg = &title
	}
	err := s.db.QueryRow(
		`INSERT INTO chat_sessions (user_id, model, title)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, title, model, created_at, updated_at`,
		userID, model, titleArg,
	).Scan(&cs.ID, &cs.UserID, &cs.Title, &cs.Model, &cs.CreatedAt, &cs.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create session: %w", err)
	}
	return &cs, nil
}

// GetSession returns a chat session, verifying it belongs to the given user.
func (s *PgStore) GetSession(userID, sessionID string) (*ChatSession, error) {
	var cs ChatSession
	err := s.db.QueryRow(
		`SELECT id, user_id, title, model, created_at, updated_at
		 FROM chat_sessions WHERE id = $1 AND user_id = $2`,
		sessionID, userID,
	).Scan(&cs.ID, &cs.UserID, &cs.Title, &cs.Model, &cs.CreatedAt, &cs.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("store: get session: %w", err)
	}
	return &cs, nil
}

// UpdateSessionTitle updates the title of a chat session, verifying ownership.
func (s *PgStore) UpdateSessionTitle(userID, sessionID, title string) error {
	res, err := s.db.Exec(
		`UPDATE chat_sessions SET title = $1, updated_at = NOW()
		 WHERE id = $2 AND user_id = $3`,
		title, sessionID, userID,
	)
	if err != nil {
		return fmt.Errorf("store: update session title: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: session not found")
	}
	return nil
}

// DeleteSession deletes a chat session, verifying ownership.
// Messages are deleted via ON DELETE CASCADE.
func (s *PgStore) DeleteSession(userID, sessionID string) error {
	res, err := s.db.Exec(
		"DELETE FROM chat_sessions WHERE id = $1 AND user_id = $2",
		sessionID, userID,
	)
	if err != nil {
		return fmt.Errorf("store: delete session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: session not found")
	}
	return nil
}

// ListMessages returns all messages for a session, ordered by created_at ASC.
func (s *PgStore) ListMessages(sessionID string) ([]ChatMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, role, content, tokens_used, cost, created_at
		 FROM chat_messages WHERE session_id = $1 ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list messages: %w", err)
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.TokensUsed, &m.Cost, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan message: %w", err)
		}
		messages = append(messages, m)
	}
	if messages == nil {
		messages = []ChatMessage{}
	}
	return messages, nil
}

// AddMessage adds a message to a chat session and updates the session's updated_at.
func (s *PgStore) AddMessage(sessionID, role, content string, tokensUsed int, cost float64) (*ChatMessage, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	var m ChatMessage
	var tokensArg *int
	var costArg *float64
	if tokensUsed > 0 {
		tokensArg = &tokensUsed
	}
	if cost > 0 {
		costArg = &cost
	}

	err = tx.QueryRow(
		`INSERT INTO chat_messages (session_id, role, content, tokens_used, cost)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, session_id, role, content, tokens_used, cost, created_at`,
		sessionID, role, content, tokensArg, costArg,
	).Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.TokensUsed, &m.Cost, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: add message: %w", err)
	}

	_, err = tx.Exec(
		"UPDATE chat_sessions SET updated_at = NOW() WHERE id = $1",
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: update session timestamp: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit: %w", err)
	}
	return &m, nil
}
