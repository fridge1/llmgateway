package payment

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler provides payment API endpoints.
type Handler struct {
	store                 store.Store
	alipay                *AlipayClient
	firstRechargeBonusCNY float64
	referralInviterBonus  float64
	referralInviteeBonus  float64
	subscribeFunc         func(userID string, planID int) error
	notifyFunc            func(userID, eventType, title, content string, refType, refID *string)
	codexNotifyFunc       func(orderNo string, callbackData []byte) error
}

// NewHandler creates a new payment Handler.
func NewHandler(s store.Store, alipay *AlipayClient, firstRechargeBonusCNY, referralInviterBonus, referralInviteeBonus float64) *Handler {
	return &Handler{
		store:                 s,
		alipay:                alipay,
		firstRechargeBonusCNY: firstRechargeBonusCNY,
		referralInviterBonus:  referralInviterBonus,
		referralInviteeBonus:  referralInviteeBonus,
	}
}

// SetNotifyFunc wires a multi-channel notifier for user-side lottery win notifications.
func (h *Handler) SetNotifyFunc(f func(userID, eventType, title, content string, refType, refID *string)) {
	h.notifyFunc = f
}

// SetSubscribeFunc sets the callback used to auto-subscribe after a subscription-linked payment.
func (h *Handler) SetSubscribeFunc(fn func(userID string, planID int) error) {
	h.subscribeFunc = fn
}

// SetCodexNotifyFunc sets the callback used to fulfill Codex (CDX-prefixed)
// orders when an Alipay notify arrives via the shared notify endpoint. Without
// this, Codex payments never update because the shared notify URL points at
// the recharge handler.
func (h *Handler) SetCodexNotifyFunc(fn func(orderNo string, callbackData []byte) error) {
	h.codexNotifyFunc = fn
}

// HandleCreate handles POST /api/payment/create.
// Creates a new payment order and returns the Alipay payment URL for browser redirect.
func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	var req struct {
		Amount     float64 `json:"amount"`
		ClientType string  `json:"client_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}

	if req.Amount < 1 || req.Amount > 10000 {
		httputil.WriteError(w, http.StatusBadRequest, "amount must be between 1 and 10000", "invalid_request", "bad_amount")
		return
	}

	// Check if user is a tenant owner
	var tenantID *string
	tenants, err := h.store.ListTenantsByUser(userID)
	if err == nil {
		for _, t := range tenants {
			if t.Role == "owner" {
				tenantID = &t.ID
				break
			}
		}
	}

	order, err := h.store.CreateOrder(userID, req.Amount, tenantID)
	if err != nil {
		slog.Error("failed to create order", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create order", "server_error", "db_error")
		return
	}

	amountStr := fmt.Sprintf("%.2f", order.Amount)
	var payURL string
	if req.ClientType == "mobile" {
		payURL, err = h.alipay.CreateWapPay(order.OrderNo, amountStr, "LLM Gateway 充值")
	} else {
		payURL, err = h.alipay.CreatePagePay(order.OrderNo, amountStr, "LLM Gateway 充值")
	}
	if err != nil {
		slog.Error("failed to create alipay trade", "error", err, "order_no", order.OrderNo)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create payment", "server_error", "alipay_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"order_no":   order.OrderNo,
		"pay_url":    payURL,
		"expired_at": order.ExpiredAt,
	})
}

// HandleRepay handles POST /api/payment/repay.
// Regenerates a payment URL for an existing pending order so the user can continue paying.
func (h *Handler) HandleRepay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	var req struct {
		OrderNo    string `json:"order_no"`
		ClientType string `json:"client_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}
	if req.OrderNo == "" {
		httputil.WriteError(w, http.StatusBadRequest, "order_no is required", "invalid_request", "missing_order_no")
		return
	}

	order, err := h.store.GetOrderByNo(req.OrderNo)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "订单不存在", "not_found", "order_not_found")
		return
	}

	if order.UserID != userID {
		httputil.WriteError(w, http.StatusForbidden, "无权操作此订单", "forbidden", "not_owner")
		return
	}

	if order.Status == "paid" {
		httputil.WriteError(w, http.StatusBadRequest, "该订单已支付", "invalid_request", "already_paid")
		return
	}
	if order.Status == "expired" || !order.ExpiredAt.After(time.Now()) {
		httputil.WriteError(w, http.StatusBadRequest, "该订单已过期，请创建新订单", "invalid_request", "order_expired")
		return
	}

	amountStr := fmt.Sprintf("%.2f", order.Amount)
	var payURL string
	if req.ClientType == "mobile" {
		payURL, err = h.alipay.CreateWapPay(order.OrderNo, amountStr, "LLM Gateway 充值")
	} else {
		payURL, err = h.alipay.CreatePagePay(order.OrderNo, amountStr, "LLM Gateway 充值")
	}
	if err != nil {
		slog.Error("failed to create alipay trade for repay", "error", err, "order_no", order.OrderNo)
		httputil.WriteError(w, http.StatusBadRequest, "支付链接已失效，请创建新订单", "payment_error", "trade_expired")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"order_no":   order.OrderNo,
		"pay_url":    payURL,
		"expired_at": order.ExpiredAt,
	})
}

// HandleAlipayNotify handles POST /api/payment/alipay/notify.
// This is called by Alipay servers — no JWT required.
func (h *Handler) HandleAlipayNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Error("failed to parse alipay notify form", "error", err)
		http.Error(w, "fail", http.StatusBadRequest)
		return
	}

	notification, err := h.alipay.VerifyNotify(r.Context(), r.PostForm)
	if err != nil {
		slog.Error("failed to verify alipay notification", "error", err)
		http.Error(w, "fail", http.StatusBadRequest)
		return
	}

	// 即时到账：成功态常见为 TRADE_SUCCESS；部分通知为 TRADE_FINISHED（交易结束）。二者择一即可入账，订单幂等防重复。
	if notification.TradeStatus != "TRADE_SUCCESS" && notification.TradeStatus != "TRADE_FINISHED" {
		slog.Info("alipay notify ignored: trade not successful", "trade_status", notification.TradeStatus, "out_trade_no", notification.OutTradeNo)
		w.Write([]byte("success"))
		return
	}

	orderNo := notification.OutTradeNo

	callbackData, _ := json.Marshal(notification)

	// Codex 代充订单（CDX 前缀）走独立的 Codex 结算回调，避免误用充值入账逻辑。
	// 共用同一个支付宝 notify URL，因此在此按订单号前缀分发。
	if strings.HasPrefix(orderNo, "CDX") {
		if h.codexNotifyFunc == nil {
			slog.Error("codex notify handler not wired, ignoring CDX order", "order_no", orderNo)
			http.Error(w, "fail", http.StatusInternalServerError)
			return
		}
		if err := h.codexNotifyFunc(orderNo, callbackData); err != nil {
			slog.Error("failed to fulfill codex order", "error", err, "order_no", orderNo)
			http.Error(w, "fail", http.StatusInternalServerError)
			return
		}
		slog.Info("codex order paid via shared notify", "order_no", orderNo)
		w.Write([]byte("success"))
		return
	}

	if err := h.store.FulfillAlipayPaidOrder(orderNo, callbackData); err != nil {
		slog.Error("failed to fulfill alipay order", "error", err, "order_no", orderNo)
		http.Error(w, "fail", http.StatusInternalServerError)
		return
	}

	// Look up order once for bonus logic below.
	order, orderErr := h.store.GetOrderByNo(orderNo)

	// Mark the "first recharge" growth task complete (best-effort, idempotent).
	if orderErr == nil && order != nil {
		if err := h.store.MarkTaskCompleted(order.UserID, "first_recharge"); err != nil {
			slog.Warn("failed to mark first_recharge task", "user_id", order.UserID, "error", err)
		}
	}

	// First recharge bonus: credit to balance.
	// 新用户首充优先级最高，不参与其他任何活动，只享受首充赠送。
	var isFirstRecharge bool
	if h.firstRechargeBonusCNY > 0 && orderErr == nil && order != nil {
		granted, err := h.store.MarkFirstRechargeGranted(order.UserID)
		if err != nil {
			slog.Warn("failed to mark first recharge bonus", "user_id", order.UserID, "error", err)
		} else if granted {
			isFirstRecharge = true
			// 按充值金额 100% 赠送（充多少送多少）
			bonus := order.Amount * h.firstRechargeBonusCNY
			if bonus > 0 {
				desc := fmt.Sprintf("首次充值赠送（充 ¥%.2f × %.0f%%）", order.Amount, h.firstRechargeBonusCNY*100)
				if err := h.store.Recharge(order.UserID, bonus, desc); err != nil {
					slog.Warn("failed to credit first recharge bonus", "user_id", order.UserID, "error", err)
				} else {
					slog.Info("first recharge bonus granted", "user_id", order.UserID, "order_amount", order.Amount, "bonus", bonus, "ratio", h.firstRechargeBonusCNY)
				}
			}
		}
	}

	// Time-bounded recharge promotion: credit a percentage of the recharge amount.
	// 仅限非首充用户参与，首充用户不参与限时活动。
	if !isFirstRecharge && orderErr == nil && order != nil {
		promo, err := h.store.GetBestActiveRechargePromotion(time.Now(), order.Amount)
		if err != nil {
			slog.Warn("failed to query recharge promotion", "user_id", order.UserID, "error", err)
		} else if promo != nil {
			bonus := math.Round(order.Amount*promo.BonusRatio*100) / 100
			if bonus > 0 {
				desc := fmt.Sprintf("充值赠送活动「%s」赠送 ¥%.2f（充 ¥%.2f × %.2f%%）",
					promo.Name, bonus, order.Amount, promo.BonusRatio*100)
				if err := h.store.Recharge(order.UserID, bonus, desc); err != nil {
					slog.Warn("failed to credit recharge promotion bonus",
						"user_id", order.UserID, "promotion_id", promo.ID, "error", err)
				} else {
					slog.Info("recharge promotion bonus granted",
						"user_id", order.UserID, "promotion_id", promo.ID,
						"order_amount", order.Amount, "bonus", bonus)
				}
			}
		}
	}

	// Referral reward: when an invited user completes their first recharge,
	// credit both the inviter and the invitee. Idempotent in the store layer.
	// Amounts come from the DB rule engine when configured, else config values.
	if orderErr == nil && order != nil {
		inviterBonus, inviteeBonus := h.referralInviterBonus, h.referralInviteeBonus
		minRecharge := 0.0
		if rule, err := h.store.GetActiveReferralRule(); err == nil {
			inviterBonus, inviteeBonus, minRecharge = rule.InviterBonusCNY, rule.InviteeBonusCNY, rule.MinFirstRechargeCNY
		}
		if (inviterBonus > 0 || inviteeBonus > 0) && order.Amount >= minRecharge {
			granted, err := h.store.GrantReferralReward(order.UserID, inviterBonus, inviteeBonus)
			if err != nil {
				slog.Warn("failed to grant referral reward", "user_id", order.UserID, "error", err)
			} else if granted {
				slog.Info("referral reward granted", "invited_user_id", order.UserID,
					"inviter_bonus", inviterBonus, "invitee_bonus", inviteeBonus)
			}
		}
	}

	// Recharge lottery: record entry and auto-draw if threshold reached (best-effort).
	if orderErr == nil && order != nil {
		if win := h.tryRechargeLottery(order.UserID, order.OrderNo, order.Amount); win != nil {
			desc := fmt.Sprintf("充值幸运奖第%d期中奖（订单 %s）", win.RoundNo, win.OrderNo)
			if err := h.store.Recharge(win.WinnerUserID, win.WinnerAmount, desc); err != nil {
				slog.Warn("recharge lottery: failed to credit prize", "user_id", win.WinnerUserID, "round", win.RoundNo, "error", err)
			} else {
				slog.Info("recharge lottery draw completed", "round", win.RoundNo, "winner", win.WinnerUserID, "amount", win.WinnerAmount)

				// 发送中奖通知（异步）
				if h.notifyFunc != nil {
					go func(w *store.RechargeLotteryWin) {
						title := "充值幸运奖中奖！"
						content := fmt.Sprintf("恭喜你在充值幸运奖第 %d 期中奖，已自动充值 %.2f 元到你的账户余额！",
							w.RoundNo, w.WinnerAmount)
						refType := "recharge_lottery"
						refID := strconv.FormatInt(int64(w.RoundNo), 10)
						h.notifyFunc(w.WinnerUserID, "recharge_lottery_win", title, content, &refType, &refID)
					}(win)
				}
			}
		}
	}

	// Lottery participation record on qualifying recharge (best-effort).
	if orderErr == nil && order != nil {
		if err := h.store.RecordLotteryParticipation(order.UserID, order.OrderNo, order.Amount); err != nil {
			slog.Warn("lottery participation record failed", "error", err, "order_no", order.OrderNo)
		}
	}

	// Auto-subscribe if this order is linked to a subscription plan.
	if orderErr == nil && order != nil && order.SubscriptionPlanID != nil && h.subscribeFunc != nil {
		if err := h.subscribeFunc(order.UserID, *order.SubscriptionPlanID); err != nil {
			slog.Error("auto-subscribe after payment failed",
				"user_id", order.UserID, "plan_id", *order.SubscriptionPlanID, "order_no", orderNo, "error", err)
		} else {
			slog.Info("auto-subscribe after payment succeeded",
				"user_id", order.UserID, "plan_id", *order.SubscriptionPlanID, "order_no", orderNo)
		}
	}

	slog.Info("alipay order fulfilled", "order_no", orderNo)
	w.Write([]byte("success"))
}

// HandleListOrders handles GET /api/payment/orders.
// Returns paginated order list for the current user.
func (h *Handler) HandleListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	orders, total, counts, err := h.store.ListOrders(userID, size, offset)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list orders", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"orders":        orders,
		"total":         total,
		"status_counts": counts,
		"page":          page,
		"size":          size,
	})
}

// tryRechargeLottery records a recharge entry and returns a winner if a draw was triggered.
// All errors are logged and swallowed — this must never block the payment flow.
func (h *Handler) tryRechargeLottery(userID, orderNo string, amount float64) *store.RechargeLotteryWin {
	lottery, err := h.store.GetActiveLottery()
	if err != nil {
		slog.Warn("recharge lottery: get active lottery failed", "error", err)
		return nil
	}
	if lottery == nil {
		return nil
	}
	if amount < 20 {
		return nil
	}
	win, err := h.store.RecordEntryAndMaybeDraw(lottery.ID, userID, orderNo, amount)
	if err != nil {
		slog.Warn("recharge lottery: record entry failed", "lottery_id", lottery.ID, "order_no", orderNo, "error", err)
		return nil
	}
	return win
}
