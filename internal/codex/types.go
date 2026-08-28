package codex

import (
	"encoding/json"
)

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	ProductID    int             `json:"product_id"`
	GuestContact json.RawMessage `json:"guest_contact"`
	ClientType   string          `json:"client_type"`
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	OrderNo       string `json:"order_no"`
	PayURL        string `json:"pay_url"`
	ExpiredAt     string `json:"expired_at"`
	ServiceWechat string `json:"service_wechat"`
}

// ShipOrderRequest 发货请求
type ShipOrderRequest struct {
	RedemptionCode string `json:"redemption_code"`
}
