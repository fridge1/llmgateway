package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/smartwalle/alipay/v3"
)

// AlipayConfig holds Alipay payment configuration.
type AlipayConfig struct {
	AppID               string
	PrivateKeyPath      string
	AlipayPublicKeyPath string
	NotifyURL           string
	ReturnURL           string
	QuitURL             string
	IsProduction        bool
}

// AlipayClient wraps the Alipay SDK client.
type AlipayClient struct {
	client    *alipay.Client
	notifyURL string
	returnURL string
	quitURL   string
}

// readKeyContent reads a PEM file and returns the base64 key content
// (strips PEM headers/footers and whitespace).
func readKeyContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read key file %s: %w", path, err)
	}
	content := string(data)
	// Strip PEM headers/footers
	for _, prefix := range []string{"-----BEGIN RSA PRIVATE KEY-----", "-----BEGIN PRIVATE KEY-----", "-----BEGIN PUBLIC KEY-----"} {
		content = strings.ReplaceAll(content, prefix, "")
	}
	for _, suffix := range []string{"-----END RSA PRIVATE KEY-----", "-----END PRIVATE KEY-----", "-----END PUBLIC KEY-----"} {
		content = strings.ReplaceAll(content, suffix, "")
	}
	content = strings.ReplaceAll(content, "\n", "")
	content = strings.ReplaceAll(content, "\r", "")
	content = strings.TrimSpace(content)
	return content, nil
}

// NewAlipayClient creates a new AlipayClient from the given configuration.
func NewAlipayClient(cfg AlipayConfig) (*AlipayClient, error) {
	privateKey, err := readKeyContent(cfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}
	client, err := alipay.New(cfg.AppID, privateKey, cfg.IsProduction)
	if err != nil {
		return nil, err
	}
	publicKey, err := readKeyContent(cfg.AlipayPublicKeyPath)
	if err != nil {
		return nil, err
	}
	if err := client.LoadAliPayPublicKey(publicKey); err != nil {
		return nil, err
	}
	return &AlipayClient{client: client, notifyURL: cfg.NotifyURL, returnURL: cfg.ReturnURL, quitURL: cfg.QuitURL}, nil
}

// CreatePagePay creates a PC website payment and returns the full Alipay payment URL.
// The user should be redirected to this URL in their browser.
func (a *AlipayClient) CreatePagePay(orderNo string, amount string, subject string) (string, error) {
	trade := alipay.TradePagePay{
		Trade: alipay.Trade{
			Subject:     subject,
			OutTradeNo:  orderNo,
			TotalAmount: amount,
			ProductCode: "FAST_INSTANT_TRADE_PAY",
			NotifyURL:   a.notifyURL,
			ReturnURL:   a.returnURL,
		},
	}
	payURL, err := a.client.TradePagePay(trade)
	if err != nil {
		return "", fmt.Errorf("alipay page pay: %w", err)
	}
	return payURL.String(), nil
}

// CreateWapPay creates a mobile H5 (WAP) payment and returns the Alipay payment URL.
func (a *AlipayClient) CreateWapPay(orderNo string, amount string, subject string) (string, error) {
	quitURL := a.quitURL
	if quitURL == "" {
		quitURL = a.returnURL
	}
	trade := alipay.TradeWapPay{
		Trade: alipay.Trade{
			Subject:     subject,
			OutTradeNo:  orderNo,
			TotalAmount: amount,
			ProductCode: "QUICK_WAP_WAY",
			NotifyURL:   a.notifyURL,
			ReturnURL:   a.returnURL,
		},
		QuitURL: quitURL,
	}
	payURL, err := a.client.TradeWapPay(trade)
	if err != nil {
		return "", fmt.Errorf("alipay wap pay: %w", err)
	}
	return payURL.String(), nil
}

// Name implements PaymentProvider.
func (a *AlipayClient) Name() string { return "alipay" }

// CreatePayment implements PaymentProvider.
func (a *AlipayClient) CreatePayment(orderNo, amount, subject string) (string, error) {
	return a.CreatePagePay(orderNo, amount, subject)
}

// VerifyCallback implements PaymentProvider.
func (a *AlipayClient) VerifyCallback(ctx context.Context, values url.Values) (*PaymentNotification, error) {
	notification, err := a.VerifyNotify(ctx, values)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(notification)
	status := ""
	if notification.TradeStatus == "TRADE_SUCCESS" || notification.TradeStatus == "TRADE_FINISHED" {
		status = "success"
	}
	return &PaymentNotification{
		OrderNo:     notification.OutTradeNo,
		TradeStatus: status,
		RawData:     raw,
	}, nil
}

// Ensure AlipayClient implements PaymentProvider at compile time.
var _ PaymentProvider = (*AlipayClient)(nil)

// VerifyNotify decodes and verifies an Alipay notification.
func (a *AlipayClient) VerifyNotify(ctx context.Context, values url.Values) (*alipay.Notification, error) {
	notification, err := a.client.DecodeNotification(ctx, values)
	if err != nil {
		return nil, err
	}
	return notification, nil
}

// RefundResult carries the upstream outcome of a refund call.
type RefundResult struct {
	TradeNo      string // Alipay trade number
	RefundAmount string // amount actually refunded (per Alipay response)
	FundChange   bool   // "Y" means funds actually moved this call
}

// Refund executes alipay.trade.refund for the given order. outRequestNo is
// the idempotency key: retries with the same value refund at most once.
func (a *AlipayClient) Refund(ctx context.Context, orderNo, refundAmount, reason, outRequestNo string) (*RefundResult, error) {
	resp, err := a.client.TradeRefund(ctx, alipay.TradeRefund{
		OutTradeNo:   orderNo,
		RefundAmount: refundAmount,
		RefundReason: reason,
		OutRequestNo: outRequestNo,
	})
	if err != nil {
		return nil, fmt.Errorf("alipay refund: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("alipay refund rejected: %s (%s)", resp.SubMsg, resp.SubCode)
	}
	return &RefundResult{
		TradeNo:      resp.TradeNo,
		RefundAmount: resp.RefundFee,
		FundChange:   resp.FundChange == "Y",
	}, nil
}

// QueryRefund fetches the refund state for reconciliation.
func (a *AlipayClient) QueryRefund(ctx context.Context, orderNo, outRequestNo string) (refunded bool, amount string, err error) {
	resp, err := a.client.TradeFastPayRefundQuery(ctx, alipay.TradeFastPayRefundQuery{
		OutTradeNo:   orderNo,
		OutRequestNo: outRequestNo,
	})
	if err != nil {
		return false, "", fmt.Errorf("alipay refund query: %w", err)
	}
	if !resp.IsSuccess() {
		return false, "", fmt.Errorf("alipay refund query rejected: %s (%s)", resp.SubMsg, resp.SubCode)
	}
	return resp.RefundStatus == "REFUND_SUCCESS", resp.RefundAmount, nil
}
