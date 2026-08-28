package codex

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/payment"
	"github.com/zhulang/llm-gateway/internal/store"
)

type AdminHandler struct {
	store  store.Store
	alipay *payment.AlipayClient
}

func NewAdminHandler(s store.Store, alipay *payment.AlipayClient) *AdminHandler {
	return &AdminHandler{store: s, alipay: alipay}
}

// HandleListOrders 管理后台订单列表
func (h *AdminHandler) HandleListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
	status := r.URL.Query().Get("status")

	orders, total, err := h.store.ListAllCodexOrders(size, offset, status)
	if err != nil {
		slog.Error("failed to list codex orders", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list orders", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders": orders,
		"total":  total,
		"page":   page,
		"size":   size,
	})
}

// HandleShipOrder 管理员发货（填写兑换码）
func (h *AdminHandler) HandleShipOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从路径提取 orderNo
	path := strings.TrimRight(r.URL.Path, "/")
	path = strings.TrimSuffix(path, "/ship")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid order number", "invalid_request", "bad_order_no")
		return
	}
	orderNo := parts[len(parts)-1]

	if orderNo == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid order number", "invalid_request", "bad_order_no")
		return
	}

	adminUserID, ok := r.Context().Value(admin.CtxUserIDKey).(string)
	if !ok || adminUserID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_required", "no_user_id")
		return
	}

	var req ShipOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}

	if strings.TrimSpace(req.RedemptionCode) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "redemption_code is required", "invalid_request", "missing_code")
		return
	}

	if err := h.store.ShipCodexOrder(orderNo, req.RedemptionCode, adminUserID); err != nil {
		slog.Error("failed to ship codex order", "error", err, "order_no", orderNo, "admin", adminUserID)
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "ship_failed", "ship_error")
		return
	}

	slog.Info("codex order shipped", "order_no", orderNo, "admin", adminUserID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

// HandleRefundOrder 管理员退款
func (h *AdminHandler) HandleRefundOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从路径提取 orderNo
	path := strings.TrimRight(r.URL.Path, "/")
	path = strings.TrimSuffix(path, "/refund")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid order number", "invalid_request", "bad_order_no")
		return
	}
	orderNo := parts[len(parts)-1]

	if orderNo == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid order number", "invalid_request", "bad_order_no")
		return
	}

	adminUserID, ok := r.Context().Value(admin.CtxUserIDKey).(string)
	if !ok || adminUserID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_required", "no_user_id")
		return
	}

	var req struct {
		Reason string  `json:"reason"`
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}

	if strings.TrimSpace(req.Reason) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "reason is required", "invalid_request", "missing_reason")
		return
	}

	// 查询订单
	order, err := h.store.GetCodexOrderByNo(orderNo)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "订单不存在", "not_found", "order_not_found")
		return
	}

	// 只能退款已支付或已发货的订单
	if order.Status != "paid" && order.Status != "shipped" {
		httputil.WriteError(w, http.StatusBadRequest, "只能退款已支付或已发货的订单", "invalid_status", "wrong_status")
		return
	}

	// 使用订单金额或指定金额
	refundAmount := req.Amount
	if refundAmount <= 0 {
		refundAmount = order.Amount
	}

	// 调用支付宝退款（outRequestNo 作为幂等键，相同键只会退款一次）
	outRequestNo := "REFUND_" + orderNo
	refundResult, err := h.alipay.Refund(r.Context(), orderNo, strconv.FormatFloat(refundAmount, 'f', 2, 64), req.Reason, outRequestNo)
	if err != nil {
		slog.Error("failed to refund codex order", "error", err, "order_no", orderNo)
		// 退款失败也要落审计记录（状态 failed）。失败记录用带时间戳的 out_request_no，
		// 避免占用成功路径的幂等键，允许后续重试。
		failedReqNo := fmt.Sprintf("REFUND_%s_FAIL_%d", orderNo, time.Now().UnixNano())
		if rerr := h.store.RecordCodexRefund(orderNo, failedReqNo, req.Reason, refundAmount, adminUserID, false, "", err.Error()); rerr != nil {
			slog.Error("failed to record codex refund(failed)", "error", rerr, "order_no", orderNo)
		}
		httputil.WriteError(w, http.StatusInternalServerError, "退款失败", "refund_failed", "alipay_error")
		return
	}

	// 退款成功：写入 codex_refunds 审计表并将订单标记为已退款（事务原子）。
	alipayTradeNo := ""
	if refundResult != nil {
		alipayTradeNo = refundResult.TradeNo
	}
	if err := h.store.RecordCodexRefund(orderNo, outRequestNo, req.Reason, refundAmount, adminUserID, true, alipayTradeNo, ""); err != nil {
		slog.Error("failed to record codex refund(succeeded)", "error", err, "order_no", orderNo)
		httputil.WriteError(w, http.StatusInternalServerError, "退款已发起但记账失败，请人工核对", "refund_record_failed", "db_error")
		return
	}

	slog.Info("codex order refunded", "order_no", orderNo, "amount", refundAmount, "admin", adminUserID, "alipay_trade_no", alipayTradeNo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

// HandleListProducts lists all Codex products including inactive (admin).
// GET /api/admin/codex/products
func (h *AdminHandler) HandleListProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	products, err := h.store.ListAllCodexProducts()
	if err != nil {
		slog.Error("failed to list codex products", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list products", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"products": products})
}

// HandleCreateProduct creates a Codex product (admin).
// POST /api/admin/codex/products
func (h *AdminHandler) HandleCreateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SKU         string  `json:"sku"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		PriceCNY    float64 `json:"price_cny"`
		SortOrder   int     `json:"sort_order"`
		Status      string  `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}
	if strings.TrimSpace(req.SKU) == "" || strings.TrimSpace(req.Name) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "sku and name are required", "invalid_request", "missing_field")
		return
	}
	if req.PriceCNY <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "price_cny must be positive", "invalid_request", "bad_price")
		return
	}
	product, err := h.store.CreateCodexProduct(req.SKU, req.Name, req.Description, req.PriceCNY, req.SortOrder, req.Status)
	if err != nil {
		slog.Error("failed to create codex product", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, err.Error(), "create_failed", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

// HandleUpdateProduct updates a Codex product (admin).
// PUT /api/admin/codex/products/{id}
func (h *AdminHandler) HandleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimRight(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid product id", "invalid_request", "bad_id")
		return
	}
	idStr := parts[len(parts)-1]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid product id", "invalid_request", "bad_id")
		return
	}
	var req struct {
		SKU         string  `json:"sku"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		PriceCNY    float64 `json:"price_cny"`
		SortOrder   int     `json:"sort_order"`
		Status      string  `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request", "bad_json")
		return
	}
	product, err := h.store.UpdateCodexProduct(id, req.SKU, req.Name, req.Description, req.PriceCNY, req.SortOrder, req.Status)
	if err != nil {
		slog.Error("failed to update codex product", "error", err, "id", id)
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "update_failed", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// HandleDeleteProduct deletes a Codex product (admin).
// DELETE /api/admin/codex/products/{id}
func (h *AdminHandler) HandleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimRight(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid product id", "invalid_request", "bad_id")
		return
	}
	idStr := parts[len(parts)-1]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid product id", "invalid_request", "bad_id")
		return
	}
	if err := h.store.DeleteCodexProduct(id); err != nil {
		slog.Error("failed to delete codex product", "error", err, "id", id)
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "delete_failed", "db_error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
