package email

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/dm"
)

// AliyunSender implements Sender using Aliyun DirectMail (邮件推送).
type AliyunSender struct {
	client    *dm.Client
	fromEmail string
	fromName  string
}

// NewAliyunSender creates an Aliyun email sender.
func NewAliyunSender(accessKey, secretKey, region, fromEmail, fromName string) (*AliyunSender, error) {
	client, err := dm.NewClientWithAccessKey(region, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("email: create aliyun dm client: %w", err)
	}

	return &AliyunSender{
		client:    client,
		fromEmail: fromEmail,
		fromName:  fromName,
	}, nil
}

// SendCode sends a verification code email using Aliyun DirectMail.
// 注意：当前使用内联HTML模板发送。如需使用阿里云控制台配置的模板，
// 需要改用 BatchSendMail 接口并预先创建收件人列表。
func (s *AliyunSender) SendCode(email, code, subject, templateID, paramKey string) error {
	request := dm.CreateSingleSendMailRequest()
	request.Scheme = "https"

	request.AccountName = s.fromEmail
	request.FromAlias = s.fromName
	request.AddressType = "1" // 1 = 随机账号
	request.ReplyToAddress = "false"
	request.ToAddress = email
	request.Subject = subject

	// 使用内联HTML模板
	// 注意：templateID 参数在当前实现中未使用，因为 SingleSendMail 不支持模板变量
	// 如需使用阿里云模板，需要改用 BatchSendMail + 预先创建的收件人列表
	htmlBody := s.buildHTMLFromTemplate(code)
	request.HtmlBody = htmlBody

	slog.Info("sending email",
		"email", email,
		"subject", subject,
	)

	response, err := s.client.SingleSendMail(request)
	if err != nil {
		return fmt.Errorf("email: aliyun send failed: %w", err)
	}

	if !response.IsSuccess() {
		return fmt.Errorf("email: aliyun api error: code=%s, message=%s",
			response.GetHttpStatus(), response.GetHttpContentString())
	}

	slog.Info("email sent successfully",
		"email", email,
		"request_id", response.RequestId,
	)

	return nil
}

// buildHTMLFromTemplate 构造邮件HTML内容
func (s *AliyunSender) buildHTMLFromTemplate(code string) string {
	tmpl := `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
    <h1 style="color: white; margin: 0;">LLM Gateway</h1>
  </div>
  <div style="background: #f7f7f7; padding: 40px; border-radius: 0 0 10px 10px;">
    <h2 style="color: #333; margin-top: 0;">验证您的邮箱地址</h2>
    <p style="color: #666; font-size: 16px;">您的验证码是：</p>
    <div style="background: white; padding: 20px; text-align: center; border-radius: 8px; margin: 20px 0;">
      <span style="font-size: 32px; font-weight: bold; color: #667eea; letter-spacing: 8px;">{{.Code}}</span>
    </div>
    <p style="color: #999; font-size: 14px;">
      本验证码 5 分钟内有效，请勿泄露给他人。<br>
      如非本人操作，请忽略此邮件。
    </p>
  </div>
  <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
    © 2026 LLM Gateway. All rights reserved.
  </div>
</body>
</html>`

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		// Fallback to simple text
		return fmt.Sprintf("<html><body><h2>您的验证码是：%s</h2><p>本验证码 5 分钟内有效。</p></body></html>", code)
	}

	var buf bytes.Buffer
	data := struct{ Code string }{Code: code}
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Sprintf("<html><body><h2>您的验证码是：%s</h2><p>本验证码 5 分钟内有效。</p></body></html>", code)
	}

	return buf.String()
}
