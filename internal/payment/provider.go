package payment

import (
	"context"
	"net/url"
)

// PaymentNotification represents a verified payment callback from any provider.
type PaymentNotification struct {
	OrderNo     string
	TradeStatus string // "success" or "finished"
	RawData     []byte // original callback data for storage
}

// PaymentProvider abstracts a payment gateway (Alipay, WeChat Pay, Stripe, etc.).
type PaymentProvider interface {
	// Name returns the provider identifier (e.g. "alipay", "wechat", "stripe").
	Name() string

	// CreatePayment creates a payment and returns a URL the user should be redirected to.
	CreatePayment(orderNo, amount, subject string) (payURL string, err error)

	// VerifyCallback verifies and parses an async payment notification.
	VerifyCallback(ctx context.Context, values url.Values) (*PaymentNotification, error)
}
