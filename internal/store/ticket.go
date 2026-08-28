package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrTicketNotFound is returned when a ticket doesn't exist or isn't visible
// to the caller.
var ErrTicketNotFound = errors.New("ticket not found")

// ticketStatuses enumerates the valid ticket workflow states.
var ticketStatuses = map[string]bool{"open": true, "pending": true, "resolved": true, "closed": true}

// ValidTicketStatus reports whether s is a known ticket status.
func ValidTicketStatus(s string) bool { return ticketStatuses[s] }

// Ticket is a user support request.
type Ticket struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	UserIdentifier string    `json:"user_identifier,omitempty"` // admin list only
	Category       string    `json:"category"`
	Subject        string    `json:"subject"`
	Status         string    `json:"status"`
	RelatedOrderNo *string   `json:"related_order_no"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TicketMessage is one entry in a ticket's conversation thread.
type TicketMessage struct {
	ID          int64           `json:"id"`
	TicketID    string          `json:"ticket_id"`
	SenderRole  string          `json:"sender_role"`
	SenderID    *string         `json:"sender_id"`
	Content     string          `json:"content"`
	Attachments json.RawMessage `json:"attachments"`
	CreatedAt   time.Time       `json:"created_at"`
}

const ticketCols = `id, user_id, category, subject, status, related_order_no, created_at, updated_at`

func scanTicket(row interface{ Scan(...any) error }, t *Ticket) error {
	return row.Scan(&t.ID, &t.UserID, &t.Category, &t.Subject, &t.Status, &t.RelatedOrderNo, &t.CreatedAt, &t.UpdatedAt)
}

// CreateTicket creates a ticket and its first message in one transaction.
func (s *PgStore) CreateTicket(userID, category, subject, content string, relatedOrderNo *string, attachments json.RawMessage) (*Ticket, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: create ticket begin: %w", err)
	}
	defer tx.Rollback()

	var t Ticket
	err = tx.QueryRow(
		`INSERT INTO tickets (user_id, category, subject, related_order_no)
		 VALUES ($1,$2,$3,$4) RETURNING `+ticketCols,
		userID, category, subject, relatedOrderNo,
	).Scan(&t.ID, &t.UserID, &t.Category, &t.Subject, &t.Status, &t.RelatedOrderNo, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create ticket: %w", err)
	}

	if attachments == nil {
		attachments = json.RawMessage(`[]`)
	}
	if _, err := tx.Exec(
		`INSERT INTO ticket_messages (ticket_id, sender_role, sender_id, content, attachments)
		 VALUES ($1,'user',$2,$3,$4)`,
		t.ID, userID, content, attachments,
	); err != nil {
		return nil, fmt.Errorf("store: create ticket message: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: create ticket commit: %w", err)
	}
	return &t, nil
}

// GetTicket returns a ticket. When userID is non-empty, visibility is
// restricted to that owner (user-facing); empty userID is the admin path.
func (s *PgStore) GetTicket(id, userID string) (*Ticket, error) {
	q := `SELECT ` + ticketCols + ` FROM tickets WHERE id=$1`
	args := []any{id}
	if userID != "" {
		q += ` AND user_id=$2`
		args = append(args, userID)
	}
	var t Ticket
	if err := scanTicket(s.db.QueryRow(q, args...), &t); err != nil {
		return nil, ErrTicketNotFound
	}
	return &t, nil
}

// AppendTicketMessage adds a message to a ticket thread and bumps updated_at.
// senderRole is "user" or "admin". Replying also flips status: an admin reply
// moves open->pending (awaiting user); a user reply moves pending->open.
func (s *PgStore) AppendTicketMessage(ticketID, senderRole, senderID, content string, attachments json.RawMessage) (*TicketMessage, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: ticket message begin: %w", err)
	}
	defer tx.Rollback()

	if attachments == nil {
		attachments = json.RawMessage(`[]`)
	}
	var m TicketMessage
	err = tx.QueryRow(
		`INSERT INTO ticket_messages (ticket_id, sender_role, sender_id, content, attachments)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, ticket_id, sender_role, sender_id, content, attachments, created_at`,
		ticketID, senderRole, senderID, content, attachments,
	).Scan(&m.ID, &m.TicketID, &m.SenderRole, &m.SenderID, &m.Content, &m.Attachments, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: ticket message insert: %w", err)
	}

	var statusFlip string
	if senderRole == "admin" {
		statusFlip = `status = CASE WHEN status = 'open' THEN 'pending' ELSE status END,`
	} else {
		statusFlip = `status = CASE WHEN status = 'pending' THEN 'open' ELSE status END,`
	}
	if _, err := tx.Exec(
		`UPDATE tickets SET `+statusFlip+` updated_at = now() WHERE id=$1`, ticketID,
	); err != nil {
		return nil, fmt.Errorf("store: ticket message touch: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: ticket message commit: %w", err)
	}
	return &m, nil
}

// ListTicketMessages returns the full thread, oldest first.
func (s *PgStore) ListTicketMessages(ticketID string) ([]TicketMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, ticket_id, sender_role, sender_id, content, attachments, created_at
		 FROM ticket_messages WHERE ticket_id=$1 ORDER BY created_at`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("store: list ticket messages: %w", err)
	}
	defer rows.Close()
	var list []TicketMessage
	for rows.Next() {
		var m TicketMessage
		if err := rows.Scan(&m.ID, &m.TicketID, &m.SenderRole, &m.SenderID, &m.Content, &m.Attachments, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan ticket message: %w", err)
		}
		list = append(list, m)
	}
	return list, nil
}

// ListUserTickets returns a user's tickets, newest first.
func (s *PgStore) ListUserTickets(userID string, limit, offset int) ([]Ticket, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM tickets WHERE user_id=$1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count user tickets: %w", err)
	}
	rows, err := s.db.Query(
		`SELECT `+ticketCols+` FROM tickets WHERE user_id=$1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list user tickets: %w", err)
	}
	defer rows.Close()
	var list []Ticket
	for rows.Next() {
		var t Ticket
		if err := scanTicket(rows, &t); err != nil {
			return nil, 0, fmt.Errorf("store: scan ticket: %w", err)
		}
		list = append(list, t)
	}
	return list, total, nil
}

// ListAdminTickets returns tickets across all users, optionally filtered by status.
func (s *PgStore) ListAdminTickets(status string, limit, offset int) ([]Ticket, int, error) {
	where := ""
	countArgs := []any{}
	if status != "" {
		where = ` WHERE t.status = $1`
		countArgs = append(countArgs, status)
	}
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM tickets t`+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count admin tickets: %w", err)
	}

	q := `SELECT t.id, t.user_id, t.category, t.subject, t.status, t.related_order_no, t.created_at, t.updated_at,
	      COALESCE(u.phone, u.email, NULLIF(u.nickname, ''), SUBSTRING(u.id::text, 1, 8))
	      FROM tickets t JOIN users u ON u.id = t.user_id` + where +
		fmt.Sprintf(` ORDER BY t.updated_at DESC LIMIT $%d OFFSET $%d`, len(countArgs)+1, len(countArgs)+2)
	rows, err := s.db.Query(q, append(countArgs, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list admin tickets: %w", err)
	}
	defer rows.Close()
	var list []Ticket
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(&t.ID, &t.UserID, &t.Category, &t.Subject, &t.Status, &t.RelatedOrderNo, &t.CreatedAt, &t.UpdatedAt, &t.UserIdentifier); err != nil {
			return nil, 0, fmt.Errorf("store: scan admin ticket: %w", err)
		}
		list = append(list, t)
	}
	return list, total, nil
}

// UpdateTicketStatus transitions a ticket to a new workflow state.
func (s *PgStore) UpdateTicketStatus(id, status string) error {
	if !ValidTicketStatus(status) {
		return fmt.Errorf("invalid ticket status %q", status)
	}
	res, err := s.db.Exec(`UPDATE tickets SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	if err != nil {
		return fmt.Errorf("store: update ticket status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrTicketNotFound
	}
	return nil
}
