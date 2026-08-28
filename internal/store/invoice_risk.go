package store

import (
	"fmt"

	"github.com/lib/pq"
)

// SetInvoiceRequestRisk stores the rule-evaluation outcome for a request.
func (s *PgStore) SetInvoiceRequestRisk(id int64, riskLevel, riskReasons string) error {
	_, err := s.db.Exec(
		`UPDATE invoice_requests SET risk_level=$2, risk_reasons=$3, updated_at=now() WHERE id=$1`,
		id, riskLevel, riskReasons)
	if err != nil {
		return fmt.Errorf("store: set invoice risk: %w", err)
	}
	return nil
}

// CountRecentInvoiceRequestsByTitle counts non-cancelled requests for the same
// title within the last N days (duplicate-application heuristic).
func (s *PgStore) CountRecentInvoiceRequestsByTitle(titleID int64, excludeRequestID int64, days int) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM invoice_requests
		 WHERE title_id=$1 AND id<>$2 AND status NOT IN ('rejected','cancelled')
		   AND created_at > now() - make_interval(days => $3)`,
		titleID, excludeRequestID, days).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: count recent invoice requests: %w", err)
	}
	return count, nil
}

// BatchUpdateInvoiceRequestStatus moves a set of pending requests to a new
// status in one statement; returns the IDs actually updated.
func (s *PgStore) BatchUpdateInvoiceRequestStatus(ids []int64, fromStatus, toStatus string) ([]int64, error) {
	rows, err := s.db.Query(
		`UPDATE invoice_requests SET status=$2, updated_at=now()
		 WHERE id = ANY($3) AND status=$1
		 RETURNING id`, fromStatus, toStatus, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("store: batch update invoice status: %w", err)
	}
	defer rows.Close()
	var updated []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: scan batch invoice id: %w", err)
		}
		updated = append(updated, id)
	}
	return updated, nil
}
