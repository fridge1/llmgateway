package notification

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler serves notification API endpoints.
type Handler struct {
	store store.Store
}

// NewHandler creates a new notification handler.
func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

// HandleList returns paginated notifications for the current user.
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	page, size := parsePagination(r)
	list, total, err := h.store.ListNotifications(userID, size, (page-1)*size)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "list_notifications")
		return
	}
	if list == nil {
		list = []store.Notification{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": list,
		"total":         total,
		"page":          page,
		"size":          size,
	})
}

// HandleUnreadCount returns the count of unread notifications.
func (h *Handler) HandleUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	count, err := h.store.CountUnreadNotifications(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "unread_count")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": count})
}

// HandleMarkRead marks a single notification as read.
func (h *Handler) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	id, ok := pathLastID(r)
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "invalid notification id", "invalid_request_error", "bad_id")
		return
	}
	if err := h.store.MarkNotificationRead(userID, id); err != nil {
		httputil.WriteError(w, http.StatusNotFound, err.Error(), "not_found", "notification")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleMarkAllRead marks all notifications as read for the current user.
func (h *Handler) HandleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	if err := h.store.MarkAllNotificationsRead(userID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "mark_all_read")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── Helpers ─────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func parsePagination(r *http.Request) (page, size int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	size, _ = strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return
}

// pathLastID extracts the last segment of the URL path as int64.
func pathLastID(r *http.Request) (int64, bool) {
	parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
	if len(parts) == 0 {
		return 0, false
	}
	// The second-to-last segment should be the ID (before /read suffix)
	for i := len(parts) - 1; i >= 0; i-- {
		id, err := strconv.ParseInt(parts[i], 10, 64)
		if err == nil && id > 0 {
			return id, true
		}
	}
	return 0, false
}
