package audit

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
)

// AuditLog represents a single audit log entry.
type AuditLog struct {
	ID          int64          `json:"id"`
	AdminUserID string         `json:"admin_user_id"`
	Action      string         `json:"action"`
	ResourceType string        `json:"resource_type"`
	ResourceID  string         `json:"resource_id"`
	Details     map[string]any `json:"details"`
	IPAddress   string         `json:"ip_address"`
	CreatedAt   time.Time      `json:"created_at"`
}

// Logger writes audit log entries to the database.
type Logger struct {
	db *sql.DB
}

// NewLogger creates an audit Logger.
func NewLogger(db *sql.DB) *Logger {
	return &Logger{db: db}
}

// Log records an audit event. Errors are logged but not returned to avoid
// blocking the main request flow.
func (l *Logger) Log(r *http.Request, action, resourceType, resourceID string, details map[string]any) {
	l.LogTenant(r, "", action, resourceType, resourceID, details)
}

// LogTenant records an audit event tagged with a tenant ID (empty = global).
// Tenant-tagged rows are visible to that tenant's owner/admin.
func (l *Logger) LogTenant(r *http.Request, tenantID, action, resourceType, resourceID string, details map[string]any) {
	adminUserID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	ip := httputil.ClientIP(r)

	detailsJSON, _ := json.Marshal(details)

	var tid any
	if tenantID != "" {
		tid = tenantID
	}
	_, err := l.db.Exec(
		`INSERT INTO admin_audit_logs (admin_user_id, action, resource_type, resource_id, details, ip_address, tenant_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		adminUserID, action, resourceType, resourceID, detailsJSON, ip, tid,
	)
	if err != nil {
		slog.Warn("failed to write audit log", "action", action, "error", err)
	}
}
