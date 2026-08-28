// Package ticket implements the user support ticket system: users open
// tickets with a message thread and optional attachments; admins reply and
// drive the status workflow. Both sides get in-app notifications.
package ticket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/audit"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

var validCategories = map[string]bool{
	"billing": true, "api": true, "account": true, "invoice": true, "other": true,
}

// Handler serves user-facing and admin ticket endpoints.
type Handler struct {
	store store.Store
	audit *audit.Logger
	// notifyFunc, when set, delivers user notifications across channels
	// (in-app + SMS per preference). Falls back to in-app only when nil.
	notifyFunc func(userID, eventType, title, content string, refType, refID *string)
}

// NewHandler creates a ticket Handler. auditLogger may be nil (no audit).
func NewHandler(s store.Store, auditLogger *audit.Logger) *Handler {
	return &Handler{store: s, audit: auditLogger}
}

// SetNotifyFunc wires a multi-channel notifier for user-side ticket updates.
func (h *Handler) SetNotifyFunc(f func(userID, eventType, title, content string, refType, refID *string)) {
	h.notifyFunc = f
}

func userID(r *http.Request) string {
	id, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	return id
}

// notifyAdmins fans an in-app notification out to every admin user.
func (h *Handler) notifyAdmins(title, content, refID string) {
	admins, err := h.store.ListAdminUsers()
	if err != nil {
		slog.Error("ticket: list admins failed", "error", err)
		return
	}
	refType := "ticket"
	notifs := make([]store.Notification, 0, len(admins))
	for _, u := range admins {
		notifs = append(notifs, store.Notification{
			UserID: u.ID, Type: "ticket", Title: title, Content: content,
			RefType: &refType, RefID: &refID,
		})
	}
	if err := h.store.BatchCreateNotifications(notifs); err != nil {
		slog.Error("ticket: notify admins failed", "error", err)
	}
}

func (h *Handler) notifyUser(uid, title, content, refID string) {
	refType := "ticket"
	if h.notifyFunc != nil {
		h.notifyFunc(uid, "ticket", title, content, &refType, &refID)
		return
	}
	if _, err := h.store.CreateNotification(uid, "ticket", title, content, &refType, &refID); err != nil {
		slog.Error("ticket: notify user failed", "error", err)
	}
}

// ---------- User endpoints ----------

// HandleTickets serves GET (list mine) / POST (create) on /api/tickets.
func (h *Handler) HandleTickets(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if uid == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "unauthorized")
		return
	}
	switch r.Method {
	case http.MethodGet:
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		size, _ := strconv.Atoi(r.URL.Query().Get("size"))
		if size < 1 || size > 100 {
			size = 20
		}
		tickets, total, err := h.store.ListUserTickets(uid, size, (page-1)*size)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to list tickets", "server_error", "db_error")
			return
		}
		if tickets == nil {
			tickets = []store.Ticket{}
		}
		writeJSON(w, map[string]any{"tickets": tickets, "total": total, "page": page, "size": size})
	case http.MethodPost:
		var req struct {
			Category       string          `json:"category"`
			Subject        string          `json:"subject"`
			Content        string          `json:"content"`
			RelatedOrderNo string          `json:"related_order_no"`
			Attachments    json.RawMessage `json:"attachments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
			return
		}
		req.Subject = strings.TrimSpace(req.Subject)
		req.Content = strings.TrimSpace(req.Content)
		if req.Subject == "" || len(req.Subject) > 200 || req.Content == "" || len(req.Content) > 10000 {
			httputil.WriteError(w, http.StatusBadRequest, "subject and content are required", "invalid_request_error", "bad_fields")
			return
		}
		if !validCategories[req.Category] {
			req.Category = "other"
		}
		var orderNo *string
		if req.RelatedOrderNo != "" {
			orderNo = &req.RelatedOrderNo
		}
		t, err := h.store.CreateTicket(uid, req.Category, req.Subject, req.Content, orderNo, req.Attachments)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to create ticket", "server_error", "db_error")
			return
		}
		h.notifyAdmins("新工单："+t.Subject, "用户提交了新工单，请及时处理。", t.ID)
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, t)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
	}
}

// HandleTicketByID serves /api/tickets/{id} (GET detail) and
// /api/tickets/{id}/messages (POST reply).
func (h *Handler) HandleTicketByID(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if uid == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "unauthorized")
		return
	}
	id, action := parseTicketPath(r.URL.Path, "/api/tickets/")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid ticket id", "invalid_request_error", "bad_id")
		return
	}

	t, err := h.store.GetTicket(id, uid)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "ticket not found", "invalid_request_error", "not_found")
		return
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		msgs, err := h.store.ListTicketMessages(t.ID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to load messages", "server_error", "db_error")
			return
		}
		if msgs == nil {
			msgs = []store.TicketMessage{}
		}
		writeJSON(w, map[string]any{"ticket": t, "messages": msgs})
	case action == "messages" && r.Method == http.MethodPost:
		if t.Status == "closed" {
			httputil.WriteError(w, http.StatusBadRequest, "ticket is closed", "invalid_request_error", "ticket_closed")
			return
		}
		var req struct {
			Content     string          `json:"content"`
			Attachments json.RawMessage `json:"attachments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
			return
		}
		req.Content = strings.TrimSpace(req.Content)
		if req.Content == "" || len(req.Content) > 10000 {
			httputil.WriteError(w, http.StatusBadRequest, "content is required", "invalid_request_error", "bad_fields")
			return
		}
		m, err := h.store.AppendTicketMessage(t.ID, "user", uid, req.Content, req.Attachments)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to reply", "server_error", "db_error")
			return
		}
		h.notifyAdmins("工单新回复："+t.Subject, "用户在工单中追加了回复。", t.ID)
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, m)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
	}
}

// ---------- Admin endpoints ----------

// HandleAdminTickets serves GET /api/admin/tickets?status=&page=&size=.
func (h *Handler) HandleAdminTickets(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	status := r.URL.Query().Get("status")
	if status != "" && !store.ValidTicketStatus(status) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid status filter", "invalid_request_error", "bad_status")
		return
	}
	tickets, total, err := h.store.ListAdminTickets(status, size, (page-1)*size)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list tickets", "server_error", "db_error")
		return
	}
	if tickets == nil {
		tickets = []store.Ticket{}
	}
	writeJSON(w, map[string]any{"tickets": tickets, "total": total, "page": page, "size": size})
}

// HandleAdminTicketByID serves /api/admin/tickets/{id} (GET detail),
// /api/admin/tickets/{id}/reply (POST), /api/admin/tickets/{id}/status (PUT).
func (h *Handler) HandleAdminTicketByID(w http.ResponseWriter, r *http.Request) {
	adminID := userID(r)
	id, action := parseTicketPath(r.URL.Path, "/api/admin/tickets/")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid ticket id", "invalid_request_error", "bad_id")
		return
	}
	t, err := h.store.GetTicket(id, "")
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "ticket not found", "invalid_request_error", "not_found")
		return
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		msgs, err := h.store.ListTicketMessages(t.ID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to load messages", "server_error", "db_error")
			return
		}
		if msgs == nil {
			msgs = []store.TicketMessage{}
		}
		writeJSON(w, map[string]any{"ticket": t, "messages": msgs})
	case action == "reply" && r.Method == http.MethodPost:
		var req struct {
			Content     string          `json:"content"`
			Attachments json.RawMessage `json:"attachments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
			return
		}
		req.Content = strings.TrimSpace(req.Content)
		if req.Content == "" || len(req.Content) > 10000 {
			httputil.WriteError(w, http.StatusBadRequest, "content is required", "invalid_request_error", "bad_fields")
			return
		}
		m, err := h.store.AppendTicketMessage(t.ID, "admin", adminID, req.Content, req.Attachments)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to reply", "server_error", "db_error")
			return
		}
		h.notifyUser(t.UserID, "工单回复："+t.Subject, "客服回复了你的工单，点击查看详情。", t.ID)
		if h.audit != nil {
			h.audit.Log(r, "ticket_reply", "ticket", t.ID, map[string]any{"subject": t.Subject})
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, m)
	case action == "status" && r.Method == http.MethodPut:
		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !store.ValidTicketStatus(req.Status) {
			httputil.WriteError(w, http.StatusBadRequest, "invalid status", "invalid_request_error", "bad_status")
			return
		}
		if err := h.store.UpdateTicketStatus(t.ID, req.Status); err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to update status", "server_error", "db_error")
			return
		}
		if req.Status == "resolved" || req.Status == "closed" {
			h.notifyUser(t.UserID, "工单状态更新："+t.Subject, "你的工单已被标记为「"+statusLabel(req.Status)+"」。", t.ID)
		}
		if h.audit != nil {
			h.audit.Log(r, "ticket_status", "ticket", t.ID, map[string]any{"status": req.Status})
		}
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
	}
}

func statusLabel(s string) string {
	switch s {
	case "resolved":
		return "已解决"
	case "closed":
		return "已关闭"
	case "pending":
		return "待用户回复"
	default:
		return "处理中"
	}
}

// parseTicketPath splits "{prefix}{id}[/{action}]" into (id, action).
func parseTicketPath(path, prefix string) (id, action string) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimRight(rest, "/")
	if rest == "" {
		return "", ""
	}
	parts := strings.SplitN(rest, "/", 2)
	id = parts[0]
	if len(parts) == 2 {
		action = parts[1]
	}
	return id, action
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
