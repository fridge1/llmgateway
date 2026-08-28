package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/httputil"
)

// Handler provides audit log API endpoints.
type Handler struct {
	db *sql.DB
}

// NewHandler creates an audit Handler.
func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// HandleList returns paginated audit logs (admin only).
// GET /api/admin/audit-logs?page=1&size=20&action=...
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	page := 1
	size := 20
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			size = n
		}
	}
	offset := (page - 1) * size
	actionFilter := r.URL.Query().Get("action")

	var total int
	var rows *sql.Rows
	var err error

	if actionFilter != "" {
		err = h.db.QueryRow(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = $1`, actionFilter).Scan(&total)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to count logs", "server_error", "db_error")
			return
		}
		rows, err = h.db.Query(
			`SELECT id, admin_user_id, action, resource_type, COALESCE(resource_id,''), details, COALESCE(ip_address,''), created_at
			 FROM admin_audit_logs WHERE action = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			actionFilter, size, offset,
		)
	} else {
		err = h.db.QueryRow(`SELECT COUNT(*) FROM admin_audit_logs`).Scan(&total)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to count logs", "server_error", "db_error")
			return
		}
		rows, err = h.db.Query(
			`SELECT id, admin_user_id, action, resource_type, COALESCE(resource_id,''), details, COALESCE(ip_address,''), created_at
			 FROM admin_audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
			size, offset,
		)
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list logs", "server_error", "db_error")
		return
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		var detailsJSON []byte
		if err := rows.Scan(&l.ID, &l.AdminUserID, &l.Action, &l.ResourceType, &l.ResourceID, &detailsJSON, &l.IPAddress, &l.CreatedAt); err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("scan error: %v", err), "server_error", "db_error")
			return
		}
		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &l.Details)
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []AuditLog{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"logs":  logs,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// HandleTenantList returns audit logs scoped to one tenant, for tenant
// owners/admins. Route: GET /api/tenants/{id}/audit?page=&size=.
// Tenant-role authorization happens in the middleware wrapping this route.
func (h *Handler) HandleTenantList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	// Path shape: /api/tenants/{uuid}/audit
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid path", "invalid_request_error", "bad_path")
		return
	}
	tenantID := parts[2]

	page := 1
	size := 20
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			size = n
		}
	}
	offset := (page - 1) * size

	var total int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM admin_audit_logs WHERE tenant_id = $1`, tenantID).Scan(&total); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to count logs", "server_error", "db_error")
		return
	}
	rows, err := h.db.Query(
		`SELECT id, admin_user_id, action, resource_type, COALESCE(resource_id,''), details, COALESCE(ip_address,''), created_at
		 FROM admin_audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, size, offset,
	)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list logs", "server_error", "db_error")
		return
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		var detailsJSON []byte
		if err := rows.Scan(&l.ID, &l.AdminUserID, &l.Action, &l.ResourceType, &l.ResourceID, &detailsJSON, &l.IPAddress, &l.CreatedAt); err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "scan error", "server_error", "db_error")
			return
		}
		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &l.Details)
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []AuditLog{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"logs":  logs,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
