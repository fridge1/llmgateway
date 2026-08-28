// Package task serves the growth-task API: an activation funnel where users
// complete onboarding milestones and claim small rewards.
package task

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler serves task endpoints.
type Handler struct {
	store store.Store
}

// NewHandler creates a task handler.
func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

// HandleList returns all active tasks with the current user's progress.
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	tasks, err := h.store.ListUserTasks(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "list_tasks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

// HandleClaim claims the reward for a completed task. Path: /api/tasks/{code}/claim
func (h *Handler) HandleClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method")
		return
	}
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}
	code := taskCodeFromPath(r.URL.Path)
	if code == "" {
		httputil.WriteError(w, http.StatusBadRequest, "task code required", "invalid_request_error", "bad_path")
		return
	}
	reward, draws, err := h.store.ClaimTaskReward(userID, code)
	switch {
	case errors.Is(err, store.ErrTaskNotCompleted):
		httputil.WriteError(w, http.StatusBadRequest, "任务尚未完成", "invalid_request_error", "not_completed")
		return
	case errors.Is(err, store.ErrTaskAlreadyClaimed):
		httputil.WriteError(w, http.StatusConflict, "奖励已领取", "invalid_request_error", "already_claimed")
		return
	case err != nil:
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error", "claim")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"reward_cny":           reward,
		"reward_lottery_draws": draws,
	})
}

// taskCodeFromPath extracts {code} from /api/tasks/{code}/claim.
func taskCodeFromPath(path string) string {
	trimmed := strings.TrimPrefix(path, "/api/tasks/")
	trimmed = strings.TrimSuffix(trimmed, "/claim")
	trimmed = strings.Trim(trimmed, "/")
	if strings.Contains(trimmed, "/") {
		return ""
	}
	return trimmed
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
