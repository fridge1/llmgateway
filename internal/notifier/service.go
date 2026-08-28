// Package notifier delivers notifications across channels: in-app always,
// SMS when the user opted in for that event type. It also runs the
// subscription-expiry reminder job.
package notifier

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/sms"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Service fans notifications out to channels based on user preferences.
type Service struct {
	store     store.Store
	cfgHolder *config.Holder
	sender    sms.Sender

	cancel context.CancelFunc
	done   chan struct{}
}

// New creates a notifier Service. sender may be nil (in-app only).
func New(s store.Store, cfgHolder *config.Holder, sender sms.Sender) *Service {
	return &Service{store: s, cfgHolder: cfgHolder, sender: sender}
}

// Notify creates the in-app notification and, if the user opted in to SMS for
// this event type and an SMS template is configured, pushes an SMS too.
// Best-effort on the SMS leg: failures are logged, never propagated.
func (s *Service) Notify(userID, eventType, title, content string, refType, refID *string) {
	if _, err := s.store.CreateNotification(userID, eventType, title, content, refType, refID); err != nil {
		slog.Error("notifier: in-app create failed", "user", userID, "type", eventType, "error", err)
		return
	}
	s.maybeSMS(userID, eventType, title)
}

func (s *Service) maybeSMS(userID, eventType, title string) {
	if s.sender == nil {
		return
	}

	cfg := s.cfgHolder.Get()

	// 根据事件类型选择短信模板
	var tpl config.SMSTemplate
	switch eventType {
	case "lottery_win", "recharge_lottery_win":
		tpl = cfg.SMS.Templates.Lottery
	case "ops_alert":
		tpl = cfg.SMS.Templates.Alert
	default:
		return // 其他事件类型不发送短信
	}

	if tpl.ID == "" {
		return // 模板未配置，跳过短信
	}

	enabled, err := s.store.SMSNotificationEnabled(userID, eventType)
	if err != nil || !enabled {
		return
	}
	user, err := s.store.GetUserByID(userID)
	if err != nil || user.Phone == "" {
		return
	}
	paramKey := tpl.ParamKey
	if paramKey == "" {
		paramKey = "name"
	}
	// Fire-and-forget: SMS latency must not block the caller.
	go func() {
		if err := s.sender.SendCode(user.Phone, title, tpl.ID, paramKey); err != nil {
			slog.Error("notifier: SMS push failed", "user", userID, "type", eventType, "error", err)
		}
	}()
}

// Start launches the subscription-expiry reminder loop (hourly tick, fires at
// 10:00 local, deduped per day per user).
func (s *Service) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go s.loop(ctx)
}

// Stop terminates the reminder loop.
func (s *Service) Stop() {
	if s.cancel == nil {
		return
	}
	s.cancel()
	<-s.done
}

func (s *Service) loop(ctx context.Context) {
	defer close(s.done)
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if time.Now().Hour() == 10 {
				s.runExpiryReminder()
			}
		case <-ctx.Done():
			return
		}
	}
}

// runExpiryReminder notifies users whose subscription expires within 3 days.
func (s *Service) runExpiryReminder() {
	subs, err := s.store.ListExpiringSubscriptions(3)
	if err != nil {
		slog.Error("notifier: expiring subscriptions query failed", "error", err)
		return
	}
	count := 0
	for _, sub := range subs {
		claimed, err := s.store.TryClaimNotificationDedup(sub.UserID, "subscription_expiry")
		if err != nil || !claimed {
			continue
		}
		days := int(time.Until(sub.ExpiresAt).Hours() / 24)
		if days < 0 {
			days = 0
		}
		s.Notify(sub.UserID, "subscription_expiry",
			"订阅即将到期",
			fmt.Sprintf("你的订阅将于 %s 到期（约 %d 天后）。到期后将恢复按量计费，及时续费可保留套餐权益。",
				sub.ExpiresAt.Format("2006-01-02"), days),
			nil, nil)
		count++
	}
	if count > 0 {
		slog.Info("notifier: subscription expiry reminders sent", "count", count)
	}
}
