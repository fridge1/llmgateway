package image

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// SessionHandler handles image session management API requests.
type SessionHandler struct {
	service *Service
}

// NewSessionHandler creates a new SessionHandler.
func NewSessionHandler(service *Service) *SessionHandler {
	return &SessionHandler{
		service: service,
	}
}

// getUserID extracts the user ID from JWT context.
func getUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(admin.CtxUserIDKey).(string)
	return userID, ok && userID != ""
}

// CreateSession handles POST /api/image/sessions
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	// Authenticate via JWT context
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	// Parse request body
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalid_request_error", "invalid_json")
		return
	}

	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "Name is required", "invalid_request_error", "missing_name")
		return
	}

	// Create session
	session := &store.ImageSession{
		UserID: userID,
		Name:   req.Name,
	}

	session, err := h.service.store.CreateImageSession(r.Context(), session)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to create session", "server_error", "create_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// GetSessions handles GET /api/image/sessions
func (h *SessionHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	// Authenticate via JWT context
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	// Get sessions
	sessions, err := h.service.store.GetImageSessions(r.Context(), userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to get sessions", "server_error", "query_failed")
		return
	}

	// Ensure we always return an array, never null
	if sessions == nil {
		sessions = []store.ImageSession{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// UpdateSession handles PUT /api/image/sessions/:id
func (h *SessionHandler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	// Authenticate via JWT context
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	// Extract session ID from path
	sessionID, err := extractSessionID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid session ID", "invalid_request_error", "invalid_session_id")
		return
	}

	// Verify ownership
	session, err := h.service.store.GetImageSession(r.Context(), sessionID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "Session not found", "invalid_request_error", "session_not_found")
		return
	}
	if session.UserID != userID {
		httputil.WriteError(w, http.StatusForbidden, "Session does not belong to user", "auth_error", "not_authorized")
		return
	}

	// Parse request body
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalid_request_error", "invalid_json")
		return
	}

	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "Name is required", "invalid_request_error", "missing_name")
		return
	}

	// Update session
	if err := h.service.store.UpdateImageSession(r.Context(), sessionID, req.Name); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to update session", "server_error", "update_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeleteSession handles DELETE /api/image/sessions/:id
func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	// Authenticate via JWT context
	userID, ok := getUserID(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	// Extract session ID from path
	sessionID, err := extractSessionID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid session ID", "invalid_request_error", "invalid_session_id")
		return
	}

	// Verify ownership
	session, err := h.service.store.GetImageSession(r.Context(), sessionID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "Session not found", "invalid_request_error", "session_not_found")
		return
	}
	if session.UserID != userID {
		httputil.WriteError(w, http.StatusForbidden, "Session does not belong to user", "auth_error", "not_authorized")
		return
	}

	// Delete session
	if err := h.service.store.DeleteImageSession(r.Context(), sessionID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to delete session", "server_error", "delete_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// extractSessionID extracts the session ID from a path like /api/image/sessions/123
func extractSessionID(path string) (int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 {
		return 0, fmt.Errorf("invalid path")
	}

	// Path is /api/image/sessions/:id or /api/image/sessions/:id/generations
	idStr := parts[3]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("invalid session ID: %w", err)
	}

	return id, nil
}
