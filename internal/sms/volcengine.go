package sms

import (
	"fmt"
	"log/slog"
	"time"

	volcsms "github.com/volcengine/volc-sdk-golang/service/sms"
)

// VolcengineSender implements Sender using the Volcengine SMS API.
type VolcengineSender struct {
	client     *volcsms.SMS
	signName   string
	smsAccount string
}

// NewVolcengineSender creates a VolcengineSender from config values.
func NewVolcengineSender(accessKey, secretKey, signName, smsAccount string) *VolcengineSender {
	instance := volcsms.NewInstance()
	instance.Client.SetAccessKey(accessKey)
	instance.Client.SetSecretKey(secretKey)
	instance.Client.SetTimeout(30 * time.Second)
	return &VolcengineSender{
		client:     instance,
		signName:   signName,
		smsAccount: smsAccount,
	}
}

// SendCode sends a verification code via Volcengine SMS.
func (s *VolcengineSender) SendCode(phone, code, templateID, paramKey string) error {
	req := &volcsms.SmsRequest{
		SmsAccount:    s.smsAccount,
		Sign:          s.signName,
		TemplateID:    templateID,
		TemplateParam: fmt.Sprintf(`{"%s":"%s"}`, paramKey, code),
		PhoneNumbers:  "86" + phone,
	}

	resp, statusCode, err := s.client.Send(req)
	if err != nil {
		slog.Error("volcengine SMS send failed",
			"phone", phone,
			"status_code", statusCode,
			"error", err,
		)
		return fmt.Errorf("SMS send failed: %w", err)
	}

	if resp.ResponseMetadata.Error != nil {
		slog.Error("volcengine SMS API error",
			"phone", phone,
			"code_id", resp.ResponseMetadata.Error.CodeN,
			"message", resp.ResponseMetadata.Error.Message,
		)
		return fmt.Errorf("SMS API error: %s", resp.ResponseMetadata.Error.Message)
	}

	slog.Info("SMS sent successfully",
		"phone", phone,
		"template_id", templateID,
		"message_id", resp.Result.MessageID,
	)
	return nil
}
