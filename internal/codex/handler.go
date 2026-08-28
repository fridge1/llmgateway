package codex

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/payment"
	"github.com/zhulang/llm-gateway/internal/store"
)

type Handler struct {
	store  store.Store
	alipay *payment.AlipayClient
}

func NewHandler(s store.Store, alipay *payment.AlipayClient) *Handler {
	return &Handler{store: s, alipay: alipay}
}

// HandleListProducts 列出 Codex 商品（公开，无需认证）
func (h *Handler) HandleListProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	products, err := h.store.ListCodexProducts()
	if err != nil {
		slog.Error("failed to list codex products", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list products", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"products": products})
}

// HandleCreateOrder 创建 Codex 订单（公开，支持游客）
func (h *Handler) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}

	// 验证游客联系方式
	if len(req.GuestContact) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "guest_contact is required", "invalid_request", "missing_contact")
		return
	}

	// 尝试解析为字符串（新格式）
	var contactStr string
	if err := json.Unmarshal(req.GuestContact, &contactStr); err == nil {
		// 是字符串格式
		if strings.TrimSpace(contactStr) == "" {
			httputil.WriteError(w, http.StatusBadRequest, "联系方式不能为空", "invalid_request", "empty_contact")
			return
		}
	} else {
		// 尝试解析为对象格式（旧格式兼容）
		var contact map[string]string
		if err := json.Unmarshal(req.GuestContact, &contact); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid guest_contact format", "invalid_request", "bad_contact")
			return
		}

		// 至少需要一个联系方式
		if contact["phone"] == "" && contact["email"] == "" && contact["wechat"] == "" {
			httputil.WriteError(w, http.StatusBadRequest, "至少提供一种联系方式（手机/微信/QQ）", "invalid_request", "no_contact")
			return
		}
	}

	// 创建订单（游客订单 userID 为 nil）
	order, err := h.store.CreateCodexOrder(req.ProductID, nil, req.GuestContact, "")
	if err != nil {
		slog.Error("failed to create codex order", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create order", "server_error", "db_error")
		return
	}

	// 创建支付宝支付
	amountStr := fmt.Sprintf("%.2f", order.Amount)
	subject := fmt.Sprintf("Codex代充 - %s", order.Product.Name)
	var payURL string

	if req.ClientType == "mobile" {
		payURL, err = h.alipay.CreateWapPay(order.OrderNo, amountStr, subject)
	} else {
		payURL, err = h.alipay.CreatePagePay(order.OrderNo, amountStr, subject)
	}

	if err != nil {
		slog.Error("failed to create alipay trade", "error", err, "order_no", order.OrderNo)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create payment", "server_error", "alipay_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateOrderResponse{
		OrderNo:       order.OrderNo,
		PayURL:        payURL,
		ExpiredAt:     order.ExpiredAt.Format("2006-01-02T15:04:05Z07:00"),
		ServiceWechat: order.ServiceWechat,
	})
}

// HandleGetOrder 查询订单详情（公开，通过订单号查询）
func (h *Handler) HandleGetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从路径提取 orderNo
	path := strings.TrimRight(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "order_no is required", "invalid_request", "missing_order_no")
		return
	}
	orderNo := parts[len(parts)-1]

	if orderNo == "" {
		httputil.WriteError(w, http.StatusBadRequest, "order_no is required", "invalid_request", "missing_order_no")
		return
	}

	order, err := h.store.GetCodexOrderByNo(orderNo)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "订单不存在", "not_found", "order_not_found")
		return
	}

	// 隐藏敏感信息
	order.CallbackData = nil

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// HandleAlipayNotify Codex 订单支付回调（支付宝异步通知）
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

	if notification.TradeStatus != "TRADE_SUCCESS" && notification.TradeStatus != "TRADE_FINISHED" {
		slog.Info("alipay notify ignored: trade not successful",
			"trade_status", notification.TradeStatus,
			"out_trade_no", notification.OutTradeNo)
		w.Write([]byte("success"))
		return
	}

	orderNo := notification.OutTradeNo

	// 只处理 Codex 订单（以 CDX 开头）
	if len(orderNo) < 3 || orderNo[:3] != "CDX" {
		slog.Info("not a codex order, ignored", "order_no", orderNo)
		w.Write([]byte("success"))
		return
	}

	callbackData, _ := json.Marshal(notification)

	if err := h.store.MarkCodexOrderPaid(orderNo, callbackData); err != nil {
		slog.Error("failed to mark codex order paid", "error", err, "order_no", orderNo)
		http.Error(w, "fail", http.StatusInternalServerError)
		return
	}

	slog.Info("codex order paid", "order_no", orderNo, "amount", notification.TotalAmount)
	w.Write([]byte("success"))
}
