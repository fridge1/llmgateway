package payment

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/audit"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// RefundHandler serves admin refund endpoints. The flow is:
// validate + insert pending (store.CreateRefund, checks refundable amount and
// user balance) → call Alipay → CompleteRefund (clawback in one DB tx) or
// FailRefund. A crash between the Alipay call and CompleteRefund leaves a
// 'pending' record that operators reconcile via QueryRefund.
type RefundHandler struct {
	store  store.Store
	alipay *AlipayClient
	audit  *audit.Logger
}

// NewRefundHandler creates a RefundHandler. alipay may be nil (refunds disabled).
func NewRefundHandler(s store.Store, alipay *AlipayClient, auditLogger *audit.Logger) *RefundHandler {
	return &RefundHandler{store: s, alipay: alipay, audit: auditLogger}
}

// HandleCreateRefund serves POST /api/admin/orders/{orderNo}/refund.
func (h *RefundHandler) HandleCreateRefund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	if h.alipay == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "alipay not configured", "server_error", "payment_disabled")
		return
	}

	// Path: /api/admin/orders/{orderNo}/refund
	path := strings.TrimSuffix(strings.TrimRight(r.URL.Path, "/"), "/refund")
	orderNo := path[strings.LastIndex(path, "/")+1:]
	if orderNo == "" || orderNo == "orders" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid order number", "invalid_request_error", "bad_order_no")
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Amount <= 0 || req.Reason == "" {
		httputil.WriteError(w, http.StatusBadRequest, "amount (>0) and reason are required", "invalid_request_error", "bad_fields")
		return
	}
	// Alipay amounts have 2 decimal places; normalise to avoid dust mismatches.
	req.Amount = float64(int64(req.Amount*100+0.5)) / 100

	operatorID, _ := r.Context().Value(admin.CtxUserIDKey).(string)

	rf, err := h.store.CreateRefund(orderNo, operatorID, req.Amount, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRefundExceedsOrder):
			httputil.WriteError(w, http.StatusBadRequest, "退款总额超过订单金额", "invalid_request_error", "refund_exceeds_order")
		case errors.Is(err, store.ErrInsufficientBalanceForRefund):
			httputil.WriteError(w, http.StatusBadRequest, "用户可用余额不足以扣回退款金额", "invalid_request_error", "insufficient_balance")
		default:
			httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "refund_rejected")
		}
		return
	}

	result, alipayErr := h.alipay.Refund(r.Context(), orderNo, fmt.Sprintf("%.2f", req.Amount), req.Reason, rf.OutRequestNo)
	if alipayErr != nil {
		slog.Error("refund: alipay call failed", "refund_id", rf.ID, "order_no", orderNo, "error", alipayErr)
		if err := h.store.FailRefund(rf.ID, alipayErr.Error()); err != nil {
			slog.Error("refund: mark failed errored", "refund_id", rf.ID, "error", err)
		}
		httputil.WriteError(w, http.StatusBadGateway, "支付宝退款失败："+alipayErr.Error(), "server_error", "alipay_refund_failed")
		return
	}

	if err := h.store.CompleteRefund(rf.ID, result.TradeNo); err != nil {
		// Money left Alipay but our books didn't update — loudest possible log;
		// the pending record remains for manual reconciliation.
		slog.Error("refund: CRITICAL complete failed after alipay success",
			"refund_id", rf.ID, "order_no", orderNo, "trade_no", result.TradeNo, "error", err)
		httputil.WriteError(w, http.StatusInternalServerError,
			"退款已到账但状态更新失败，请勿重试，联系技术核对（refund_id: "+rf.ID+"）", "server_error", "refund_settle_failed")
		return
	}

	if h.audit != nil {
		h.audit.Log(r, "order_refund", "order", orderNo, map[string]any{
			"refund_id": rf.ID, "amount": req.Amount, "reason": req.Reason, "trade_no": result.TradeNo,
		})
	}
	if _, err := h.store.CreateNotification(rf.UserID, "refund",
		"退款成功", fmt.Sprintf("你的订单 %s 已退款 ¥%.2f，款项将原路退回支付宝。", orderNo, req.Amount), nil, nil); err != nil {
		slog.Error("refund: notify user failed", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok", "refund_id": rf.ID, "trade_no": result.TradeNo})
}

// HandleListRefunds serves GET /api/admin/refunds?page=&size=.
func (h *RefundHandler) HandleListRefunds(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	refunds, total, err := h.store.ListRefunds(size, (page-1)*size)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list refunds", "server_error", "db_error")
		return
	}
	if refunds == nil {
		refunds = []store.Refund{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"refunds": refunds, "total": total, "page": page, "size": size})
}
