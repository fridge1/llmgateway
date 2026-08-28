package lottery

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/store"
)

type Handler struct {
	store      store.Store
	notifyFunc func(userID, eventType, title, content string, refType, refID *string)
}

func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

// SetNotifyFunc 注入通知服务
func (h *Handler) SetNotifyFunc(f func(userID, eventType, title, content string, refType, refID *string)) {
	h.notifyFunc = f
}

// HandleAdminListEvents lists all lottery events.
// GET /api/admin/lottery/events
func (h *Handler) HandleAdminListEvents(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	events, total, err := h.store.ListLotteryEvents(size, offset)
	if err != nil {
		slog.Error("list lottery events failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"events": events,
		"total":  total,
	})
}

// HandleAdminCreateEvent creates a new lottery event.
// POST /api/admin/lottery/events
func (h *Handler) HandleAdminCreateEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                string  `json:"name"`
		Description         string  `json:"description"`
		Status              string  `json:"status"`
		MinRechargeCNY      float64 `json:"min_recharge_cny"`
		MinOrderCountToDraw int     `json:"min_order_count_to_draw"`
		StartTime           *string `json:"start_time"`
		EndTime             *string `json:"end_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.MinOrderCountToDraw < 0 {
		http.Error(w, "invalid min_order_count_to_draw: must be >= 0", http.StatusBadRequest)
		return
	}

	var startTime, endTime *time.Time
	if req.StartTime != nil && *req.StartTime != "" {
		t, err := time.Parse(time.RFC3339, *req.StartTime)
		if err != nil {
			http.Error(w, "invalid start_time", http.StatusBadRequest)
			return
		}
		startTime = &t
	}
	if req.EndTime != nil && *req.EndTime != "" {
		t, err := time.Parse(time.RFC3339, *req.EndTime)
		if err != nil {
			http.Error(w, "invalid end_time", http.StatusBadRequest)
			return
		}
		endTime = &t
	}

	event, err := h.store.CreateLotteryEvent(req.Name, req.Description, req.Status, req.MinRechargeCNY, req.MinOrderCountToDraw, startTime, endTime)
	if err != nil {
		slog.Error("create lottery event failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// HandleAdminUpdateEvent updates a lottery event.
// PUT /api/admin/lottery/events/{id}
func (h *Handler) HandleAdminUpdateEvent(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/lottery/events/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name                string  `json:"name"`
		Description         string  `json:"description"`
		Status              string  `json:"status"`
		MinRechargeCNY      float64 `json:"min_recharge_cny"`
		MinOrderCountToDraw int     `json:"min_order_count_to_draw"`
		StartTime           *string `json:"start_time"`
		EndTime             *string `json:"end_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.MinOrderCountToDraw < 0 {
		http.Error(w, "invalid min_order_count_to_draw: must be >= 0", http.StatusBadRequest)
		return
	}

	var startTime, endTime *time.Time
	if req.StartTime != nil && *req.StartTime != "" {
		t, err := time.Parse(time.RFC3339, *req.StartTime)
		if err != nil {
			http.Error(w, "invalid start_time", http.StatusBadRequest)
			return
		}
		startTime = &t
	}
	if req.EndTime != nil && *req.EndTime != "" {
		t, err := time.Parse(time.RFC3339, *req.EndTime)
		if err != nil {
			http.Error(w, "invalid end_time", http.StatusBadRequest)
			return
		}
		endTime = &t
	}

	event, err := h.store.UpdateLotteryEvent(id, req.Name, req.Description, req.Status, req.MinRechargeCNY, req.MinOrderCountToDraw, startTime, endTime)
	if err != nil {
		// 业务校验错误（非法状态、禁止恢复已结束活动）返回 400
		if strings.HasPrefix(err.Error(), "store:") && (strings.Contains(err.Error(), "invalid lottery status") || strings.Contains(err.Error(), "cannot re-activate")) {
			slog.Warn("update lottery event rejected", "error", err, "event_id", id)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.Error("update lottery event failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// HandleAdminListPrizes lists prizes for an event.
// GET /api/admin/lottery/events/{id}/prizes
func (h *Handler) HandleAdminListPrizes(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/lottery/events/")
	idStr = strings.TrimSuffix(idStr, "/prizes")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	prizes, err := h.store.ListLotteryPrizes(id)
	if err != nil {
		slog.Error("list lottery prizes failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"prizes": prizes})
}

// HandleAdminCreatePrize creates a prize for an event.
// POST /api/admin/lottery/events/{id}/prizes
func (h *Handler) HandleAdminCreatePrize(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/lottery/events/")
	idStr = strings.TrimSuffix(idStr, "/prizes")
	eventID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Weight      int     `json:"weight"`
		TotalStock  int     `json:"total_stock"`
		PrizeType   string  `json:"prize_type"`
		PrizeValue  float64 `json:"prize_value"`
		SortOrder   int     `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	prize, err := h.store.CreateLotteryPrize(eventID, req.Name, req.Description, req.Weight, req.TotalStock, req.PrizeType, req.PrizeValue, req.SortOrder)
	if err != nil {
		slog.Error("create lottery prize failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prize)
}

// HandleAdminUpdatePrize updates a prize.
// PUT /api/admin/lottery/prizes/{id}
func (h *Handler) HandleAdminUpdatePrize(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/lottery/prizes/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Weight      int     `json:"weight"`
		TotalStock  int     `json:"total_stock"`
		PrizeType   string  `json:"prize_type"`
		PrizeValue  float64 `json:"prize_value"`
		SortOrder   int     `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	prize, err := h.store.UpdateLotteryPrize(id, req.Name, req.Description, req.Weight, req.TotalStock, req.PrizeType, req.PrizeValue, req.SortOrder)
	if err != nil {
		slog.Error("update lottery prize failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prize)
}

// HandleAdminDeletePrize deletes a prize.
// DELETE /api/admin/lottery/prizes/{id}
func (h *Handler) HandleAdminDeletePrize(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/lottery/prizes/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteLotteryPrize(id); err != nil {
		slog.Error("delete lottery prize failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleAdminListRecords lists draw records for an event.
// GET /api/admin/lottery/events/{id}/records
func (h *Handler) HandleAdminListRecords(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/lottery/events/")
	idStr = strings.TrimSuffix(idStr, "/records")
	eventID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	records, total, err := h.store.ListLotteryRecords(eventID, size, offset)
	if err != nil {
		slog.Error("list lottery records failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"records": records,
		"total":   total,
	})
}

// HandleUserGetCurrent gets the active lottery event and its prizes (user-facing).
// GET /api/lottery/current
func (h *Handler) HandleUserGetCurrent(w http.ResponseWriter, r *http.Request) {
	event, prizes, err := h.store.GetActiveLotteryInfo()
	if err != nil {
		slog.Error("get active lottery info failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Hide weight from user response.
	type userPrize struct {
		ID             int     `json:"id"`
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		TotalStock     int     `json:"total_stock"`
		RemainingStock int     `json:"remaining_stock"`
		PrizeType      string  `json:"prize_type"`
		PrizeValue     float64 `json:"prize_value"`
	}
	var userPrizes []userPrize
	for _, p := range prizes {
		userPrizes = append(userPrizes, userPrize{
			ID:             p.ID,
			Name:           p.Name,
			Description:    p.Description,
			TotalStock:     p.TotalStock,
			RemainingStock: p.RemainingStock,
			PrizeType:      p.PrizeType,
			PrizeValue:     p.PrizeValue,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"event":  event,
		"prizes": userPrizes,
	})
}

// HandleUserGetRecords gets public lottery winners across all events.
// GET /api/lottery/records
func (h *Handler) HandleUserGetRecords(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	records, total, err := h.store.ListAllPublicLotteryWinners(size, offset)
	if err != nil {
		slog.Error("list public lottery winners failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"records": records,
		"total":   total,
	})
}

// HandleUserGetMyRecords returns the current user's own lottery participation
// and winning records across all events (JWT required).
// GET /api/lottery/my-records
func (h *Handler) HandleUserGetMyRecords(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	records, total, err := h.store.ListUserLotteryRecords(userID, size, offset)
	if err != nil {
		slog.Error("list user lottery records failed", "error", err, "user_id", userID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"records": records,
		"total":   total,
	})
}


// HandleAdminDrawEvent performs manual lottery draw for an event.
// POST /api/admin/lottery/events/{id}/draw
func (h *Handler) HandleAdminDrawEvent(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/lottery/events/")
	idStr = strings.TrimSuffix(idStr, "/draw")
	eventID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}

	// 获取活动信息
	event, err := h.store.GetLotteryEvent(eventID)
	if err != nil {
		slog.Error("get lottery event failed", "event_id", eventID, "error", err)
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}

	// 获取所有参与者（开奖前）
	allParticipants, _, err := h.store.ListLotteryRecords(eventID, 10000, 0)
	if err != nil {
		slog.Error("get participants failed", "event_id", eventID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 执行开奖（传入操作人 ID 用于幂等审计）
	drawnBy, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	winners, err := h.store.DrawEventLottery(eventID, drawnBy)
	if err != nil {
		slog.Error("draw event lottery failed", "error", err, "event_id", eventID)
		// 区分幂等冲突与系统错误：已开奖/非 active 属业务错误，返回 409
		if strings.Contains(err.Error(), "already drawn") || strings.Contains(err.Error(), "is not active") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		// 未达到最低开奖笔数：业务错误，返回 400
		if strings.Contains(err.Error(), "lottery: cannot draw event") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("lottery draw completed", "event_id", eventID, "winners_count", len(winners), "drawn_by", drawnBy)

	// 异步发送通知
	if h.notifyFunc != nil && len(allParticipants) > 0 {
		go h.notifyLotteryResults(event, winners, allParticipants)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"winners": winners,
		"count":   len(winners),
	})
}

// notifyLotteryResults 批量发送抽奖结果通知（中奖 + 未中奖）
func (h *Handler) notifyLotteryResults(event *store.LotteryEvent, winners []store.LotteryRecord, allParticipants []store.LotteryRecord) {
	// 构建中奖者 map（快速查找）
	winnerMap := make(map[string]bool, len(winners))
	for _, w := range winners {
		winnerMap[w.UserID] = true
	}

	// Worker pool 限制并发（避免 DB 连接池耗尽）
	const maxWorkers = 20
	sem := make(chan struct{}, maxWorkers)

	// 通知所有参与者
	for _, p := range allParticipants {
		sem <- struct{}{} // 获取信号量

		go func(participant store.LotteryRecord) {
			defer func() { <-sem }() // 释放信号量

			if winnerMap[participant.UserID] {
				// 中奖通知
				h.sendWinnerNotification(event, participant)
			} else {
				// 未中奖通知
				h.sendLoserNotification(event, participant.UserID)
			}
		}(p)
	}

	// 等待所有 worker 完成
	for i := 0; i < maxWorkers; i++ {
		sem <- struct{}{}
	}
}

// sendWinnerNotification 发送中奖通知
func (h *Handler) sendWinnerNotification(event *store.LotteryEvent, record store.LotteryRecord) {
	title := "恭喜中奖！"

	var content string
	switch record.PrizeType {
	case "balance":
		content = fmt.Sprintf("恭喜你在「%s」中抽中 %s！%.2f 元已充值到你的账户余额。",
			event.Name, record.PrizeName, record.PrizeValue)
	case "match_recharge":
		content = fmt.Sprintf("恭喜你在「%s」中抽中 %s！%.2f 元已充值到你的账户余额。",
			event.Name, record.PrizeName, record.PrizeValue)
	case "physical":
		content = fmt.Sprintf("恭喜你在「%s」中抽中 %s！我们将尽快安排发货，请留意站内消息或短信通知。",
			event.Name, record.PrizeName)
	default:
		content = fmt.Sprintf("恭喜你在「%s」中抽中 %s！",
			event.Name, record.PrizeName)
	}

	refType := "lottery_event"
	refID := strconv.Itoa(event.ID)
	h.notifyFunc(record.UserID, "lottery_win", title, content, &refType, &refID)
}

// sendLoserNotification 发送未中奖通知
func (h *Handler) sendLoserNotification(event *store.LotteryEvent, userID string) {
	title := "抽奖结果通知"
	content := fmt.Sprintf("感谢你参与「%s」抽奖活动，很遗憾这次没有中奖。期待下次好运！", event.Name)

	refType := "lottery_event"
	refID := strconv.Itoa(event.ID)
	h.notifyFunc(userID, "lottery_lose", title, content, &refType, &refID)
}

