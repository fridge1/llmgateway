package chat

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler provides chat session API endpoints.
type Handler struct {
	store store.Store
}

// NewHandler creates a new chat Handler.
func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

// HandleListSessions returns all chat sessions for the authenticated user.
// GET /api/chat/sessions
func (h *Handler) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	sessions, err := h.store.ListSessions(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list sessions", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"sessions": sessions})
}

type createSessionRequest struct {
	Model string `json:"model"`
	Title string `json:"title"`
}

// HandleCreateSession creates a new chat session.
// POST /api/chat/sessions
func (h *Handler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.Model == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
		return
	}

	session, err := h.store.CreateSession(userID, req.Model, req.Title)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create session", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

// extractSessionID extracts the session ID from a URL path like /api/chat/sessions/{id} or /api/chat/sessions/{id}/messages.
func extractSessionID(path string) string {
	trimmed := strings.TrimPrefix(path, "/api/chat/sessions/")
	parts := strings.Split(trimmed, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// HandleGetMessages returns all messages for a chat session.
// GET /api/chat/sessions/{id}/messages
func (h *Handler) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing session id", "invalid_request_error", "missing_id")
		return
	}

	// Verify ownership.
	if _, err := h.store.GetSession(userID, sessionID); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "session not found", "invalid_request_error", "not_found")
		return
	}

	messages, err := h.store.ListMessages(sessionID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list messages", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"messages": messages})
}

type updateSessionRequest struct {
	Title string `json:"title"`
}

// HandleUpdateSession updates a chat session's title.
// PUT /api/chat/sessions/{id}
func (h *Handler) HandleUpdateSession(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing session id", "invalid_request_error", "missing_id")
		return
	}

	var req updateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	if err := h.store.UpdateSessionTitle(userID, sessionID, req.Title); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "session not found", "invalid_request_error", "not_found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// HandleDeleteSession deletes a chat session.
// DELETE /api/chat/sessions/{id}
func (h *Handler) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing session id", "invalid_request_error", "missing_id")
		return
	}

	if err := h.store.DeleteSession(userID, sessionID); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "session not found", "invalid_request_error", "not_found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

type addMessageRequest struct {
	Role       string  `json:"role"`
	Content    string  `json:"content"`
	TokensUsed int     `json:"tokens_used"`
	Cost       float64 `json:"cost"`
}

// HandleAddMessage adds a message to a chat session.
// POST /api/chat/sessions/{id}/messages
func (h *Handler) HandleAddMessage(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing session id", "invalid_request_error", "missing_id")
		return
	}

	// Verify ownership.
	if _, err := h.store.GetSession(userID, sessionID); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "session not found", "invalid_request_error", "not_found")
		return
	}

	var req addMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.Role == "" || strings.TrimSpace(req.Content) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "role and content are required", "invalid_request_error", "missing_fields")
		return
	}

	msg, err := h.store.AddMessage(sessionID, req.Role, req.Content, req.TokensUsed, req.Cost)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to add message", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}
