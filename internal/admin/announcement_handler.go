package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// AnnouncementHandler provides admin endpoints for managing announcements.
type AnnouncementHandler struct {
	store store.Store
}

// NewAnnouncementHandler creates an AnnouncementHandler.
func NewAnnouncementHandler(s store.Store) *AnnouncementHandler {
	return &AnnouncementHandler{store: s}
}

// HandleList returns paginated list of all announcements.
// GET /api/admin/announcements?page=1&size=20
func (h *AnnouncementHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	list, total, err := h.store.ListAnnouncements(size, offset)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list announcements", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"announcements": list,
		"total":         total,
		"page":          page,
		"size":          size,
	})
}

type createAnnouncementRequest struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	DisplayMode string `json:"display_mode"`
}

// HandleCreate creates a new announcement.
// POST /api/admin/announcements
func (h *AnnouncementHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	adminID, _ := r.Context().Value(CtxUserIDKey).(string)

	var req createAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.Title == "" {
		httputil.WriteError(w, http.StatusBadRequest, "title is required", "invalid_request_error", "missing_title")
		return
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}
	if req.DisplayMode == "" {
		req.DisplayMode = "banner"
	}

	a, err := h.store.CreateAnnouncement(req.Title, req.Content, req.Status, req.Priority, req.DisplayMode, adminID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create announcement", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

// HandleUpdate updates an existing announcement.
// PUT /api/admin/announcements/{id}
func (h *AnnouncementHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := extractAnnouncementID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid announcement id", "invalid_request_error", "bad_id")
		return
	}

	var req createAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	existing, err := h.store.GetAnnouncementByID(id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "announcement not found", "invalid_request_error", "not_found")
		return
	}
	if req.Title == "" {
		req.Title = existing.Title
	}
	if req.Content == "" {
		req.Content = existing.Content
	}
	if req.Status == "" {
		req.Status = existing.Status
	}
	if req.Priority == "" {
		req.Priority = existing.Priority
	}
	if req.DisplayMode == "" {
		req.DisplayMode = existing.DisplayMode
	}

	a, err := h.store.UpdateAnnouncement(id, req.Title, req.Content, req.Status, req.Priority, req.DisplayMode)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update announcement", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

// HandleDelete deletes an announcement.
// DELETE /api/admin/announcements/{id}
func (h *AnnouncementHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := extractAnnouncementID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid announcement id", "invalid_request_error", "bad_id")
		return
	}

	if err := h.store.DeleteAnnouncement(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.WriteError(w, http.StatusNotFound, "announcement not found", "invalid_request_error", "not_found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete announcement", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleListPublished returns all published announcements for user-facing display.
// GET /api/announcements
func (h *AnnouncementHandler) HandleListPublished(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListPublishedAnnouncements()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list announcements", "server_error", "db_error")
		return
	}
	if list == nil {
		list = []store.Announcement{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"announcements": list,
	})
}

func extractAnnouncementID(path string) (int64, error) {
	path = strings.TrimRight(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return 0, fmt.Errorf("invalid path")
	}
	return strconv.ParseInt(parts[len(parts)-1], 10, 64)
}
