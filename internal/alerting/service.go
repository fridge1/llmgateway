// Package alerting runs a background loop that watches gateway health signals
// (metric counter deltas and DB reachability) against DB-configured threshold
// rules, and notifies admins via in-app notification and SMS. Alert events are
// recorded in alert_events, which doubles as the cooldown dedup record so a
// noisy signal cannot spam admins.
package alerting

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/metrics"
	"github.com/zhulang/llm-gateway/internal/sms"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Service owns the alerting background goroutine.
type Service struct {
	store     store.Store
	db        *sql.DB
	cfgHolder *config.Holder
	sender    sms.Sender

	// last observed counter values, keyed by metric name (delta-based rules).
	last map[string]int64

	cancel context.CancelFunc
	done   chan struct{}
}

// New creates an alerting Service. Call Start to launch the background loop.
// db is used for the db_health probe; sender may be nil (in-app only).
func New(s store.Store, db *sql.DB, cfgHolder *config.Holder, sender sms.Sender) *Service {
	return &Service{store: s, db: db, cfgHolder: cfgHolder, sender: sender, last: make(map[string]int64)}
}

// Start launches the evaluation loop.
func (s *Service) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	// Prime baselines so a restart doesn't alert on counters accumulated before.
	s.prime()
	go s.loop(ctx)
}

// Stop signals the background goroutine to exit and waits for it.
func (s *Service) Stop() {
	if s.cancel == nil {
		return
	}
	s.cancel()
	<-s.done
}

func (s *Service) interval() time.Duration {
	if d := s.cfgHolder.Get().Alerting.CheckInterval; d > 0 {
		return d
	}
	return time.Minute
}

func (s *Service) loop(ctx context.Context) {
	defer close(s.done)
	ticker := time.NewTicker(s.interval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.Evaluate(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// counterValue reads the current value of a delta-monitored metric.
// Unknown metrics return -1 so their rules are skipped gracefully.
func counterValue(metric string) int64 {
	m := metrics.Get()
	switch metric {
	case "circuit_opened":
		return m.CircuitOpened.Load()
	case "billing_jobs_dropped":
		return m.BillingJobsDropped.Load()
	case "billing_queue_overflow":
		return m.BillingQueueOverflow.Load()
	case "upstream_failures":
		return m.UpstreamFailures.Load()
	default:
		return -1
	}
}

// prime records current counter values as the delta baseline.
func (s *Service) prime() {
	for _, metric := range []string{"circuit_opened", "billing_jobs_dropped", "billing_queue_overflow", "upstream_failures"} {
		if v := counterValue(metric); v >= 0 {
			s.last[metric] = v
		}
	}
}

// Evaluate runs one round of rule checks. Exported for tests.
func (s *Service) Evaluate(ctx context.Context) {
	if !s.cfgHolder.Get().Alerting.Enabled {
		return
	}
	rules, err := s.store.ListAlertRules()
	if err != nil {
		slog.Error("alerting: list rules failed", "error", err)
		return
	}
	for _, rule := range rules {
		if !rule.Enabled {
			// Keep baselines moving even when disabled, so re-enabling
			// doesn't fire on the backlog accumulated while off.
			if v := counterValue(rule.Metric); v >= 0 {
				s.last[rule.Metric] = v
			}
			continue
		}
		switch rule.Metric {
		case "db_health":
			s.checkDBHealth(ctx, rule)
		default:
			s.checkCounterDelta(rule)
		}
	}
}

func (s *Service) checkDBHealth(ctx context.Context, rule store.AlertRule) {
	if s.db == nil {
		return
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.db.PingContext(pingCtx); err != nil {
		msg := fmt.Sprintf("数据库健康检查失败：%v", err)
		s.fire(rule, msg, 1)
	}
}

func (s *Service) checkCounterDelta(rule store.AlertRule) {
	cur := counterValue(rule.Metric)
	if cur < 0 {
		return
	}
	prev, ok := s.last[rule.Metric]
	s.last[rule.Metric] = cur
	if !ok {
		return
	}
	delta := cur - prev
	if delta >= rule.Threshold {
		msg := fmt.Sprintf("%s：最近 %s 内发生 %d 次（阈值 %d）",
			rule.DisplayName, s.interval().String(), delta, rule.Threshold)
		s.fire(rule, msg, delta)
	}
}

// fire records the alert (cooldown-deduped in the DB) and notifies admins.
func (s *Service) fire(rule store.AlertRule, message string, value int64) {
	claimed, err := s.store.TryClaimAlertEvent(rule.Metric, message, value, rule.Threshold, rule.CooldownSeconds)
	if err != nil {
		slog.Error("alerting: claim event failed", "metric", rule.Metric, "error", err)
		return
	}
	if !claimed {
		return // within cooldown window
	}
	slog.Warn("alerting: alert fired", "metric", rule.Metric, "message", message)

	admins, err := s.store.ListAdminUsers()
	if err != nil {
		slog.Error("alerting: list admins failed", "error", err)
		return
	}

	title := "运维告警：" + rule.DisplayName
	notifs := make([]store.Notification, 0, len(admins))
	for _, u := range admins {
		notifs = append(notifs, store.Notification{
			UserID:  u.ID,
			Type:    "ops_alert",
			Title:   title,
			Content: message,
		})
	}
	if err := s.store.BatchCreateNotifications(notifs); err != nil {
		slog.Error("alerting: create notifications failed", "error", err)
	}

	s.sendSMS(admins, rule)
}

// sendSMS pushes the alert to admin phones. The alert SMS template carries a
// single variable (the rule display name), matching the SendCode contract.
func (s *Service) sendSMS(admins []store.User, rule store.AlertRule) {
	cfg := s.cfgHolder.Get()
	tpl := cfg.SMS.Templates.Alert
	if s.sender == nil || tpl.ID == "" {
		return
	}
	paramKey := tpl.ParamKey
	if paramKey == "" {
		paramKey = "name"
	}
	// Prefer explicitly configured phones; fall back to admin users' phones.
	phones := cfg.Alerting.AdminPhones
	if len(phones) == 0 {
		for _, u := range admins {
			if u.Phone != "" {
				phones = append(phones, u.Phone)
			}
		}
	}
	for _, phone := range phones {
		if err := s.sender.SendCode(phone, rule.DisplayName, tpl.ID, paramKey); err != nil {
			slog.Error("alerting: alert SMS failed", "phone", phone, "error", err)
		}
	}
}
