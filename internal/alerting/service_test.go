package alerting

import (
	"context"
	"testing"
	"time"

	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/metrics"
	"github.com/zhulang/llm-gateway/internal/store"
)

// stubStore implements just the Store methods the alerting service touches.
type stubStore struct {
	store.Store // embed to satisfy the interface; unused methods panic if called

	rules   []store.AlertRule
	claimed []string // metrics that fired
	claimOK bool     // whether TryClaimAlertEvent grants the claim
	notifs  []store.Notification
}

func (s *stubStore) ListAlertRules() ([]store.AlertRule, error) { return s.rules, nil }

func (s *stubStore) TryClaimAlertEvent(metric, message string, value, threshold int64, cooldownSeconds int) (bool, error) {
	if !s.claimOK {
		return false, nil
	}
	s.claimed = append(s.claimed, metric)
	return true, nil
}

func (s *stubStore) ListAdminUsers() ([]store.User, error) {
	return []store.User{{ID: "admin-1", Phone: "13800000000"}}, nil
}

func (s *stubStore) BatchCreateNotifications(notifs []store.Notification) error {
	s.notifs = append(s.notifs, notifs...)
	return nil
}

func newTestService(st *stubStore, enabled bool) *Service {
	cfg := &config.Config{}
	cfg.Alerting.Enabled = enabled
	cfg.Alerting.CheckInterval = time.Minute
	return New(st, nil, config.NewHolder(cfg), nil)
}

func TestEvaluateFiresOnCounterDelta(t *testing.T) {
	st := &stubStore{
		claimOK: true,
		rules: []store.AlertRule{
			{ID: 1, Metric: "circuit_opened", DisplayName: "上游熔断触发", Threshold: 1, CooldownSeconds: 60, Enabled: true},
		},
	}
	svc := newTestService(st, true)
	svc.prime()

	// First round: no delta, no alert.
	svc.Evaluate(context.Background())
	if len(st.claimed) != 0 {
		t.Fatalf("expected no alert before counter moves, got %v", st.claimed)
	}

	// Counter moves past threshold.
	metrics.Get().CircuitOpened.Add(2)
	svc.Evaluate(context.Background())
	if len(st.claimed) != 1 || st.claimed[0] != "circuit_opened" {
		t.Fatalf("expected circuit_opened alert, got %v", st.claimed)
	}
	if len(st.notifs) != 1 || st.notifs[0].UserID != "admin-1" || st.notifs[0].Type != "ops_alert" {
		t.Fatalf("expected one ops_alert notification for admin-1, got %+v", st.notifs)
	}

	// Baseline advanced: no further delta, no repeat alert.
	st.claimed = nil
	svc.Evaluate(context.Background())
	if len(st.claimed) != 0 {
		t.Fatalf("expected no alert without new delta, got %v", st.claimed)
	}
}

func TestEvaluateRespectsCooldownClaim(t *testing.T) {
	st := &stubStore{
		claimOK: false, // simulate cooldown window: claim denied
		rules: []store.AlertRule{
			{ID: 1, Metric: "billing_jobs_dropped", DisplayName: "计费结算任务丢弃", Threshold: 1, CooldownSeconds: 60, Enabled: true},
		},
	}
	svc := newTestService(st, true)
	svc.prime()

	metrics.Get().BillingJobsDropped.Add(1)
	svc.Evaluate(context.Background())
	if len(st.notifs) != 0 {
		t.Fatalf("expected no notification when claim denied by cooldown, got %+v", st.notifs)
	}
}

func TestEvaluateSkipsWhenDisabled(t *testing.T) {
	st := &stubStore{
		claimOK: true,
		rules: []store.AlertRule{
			{ID: 1, Metric: "circuit_opened", DisplayName: "上游熔断触发", Threshold: 1, CooldownSeconds: 60, Enabled: true},
		},
	}
	svc := newTestService(st, false) // alerting disabled in config
	svc.prime()

	metrics.Get().CircuitOpened.Add(1)
	svc.Evaluate(context.Background())
	if len(st.claimed) != 0 {
		t.Fatalf("expected no alert when alerting disabled, got %v", st.claimed)
	}
}

func TestDisabledRuleAdvancesBaseline(t *testing.T) {
	st := &stubStore{
		claimOK: true,
		rules: []store.AlertRule{
			{ID: 1, Metric: "upstream_failures", DisplayName: "上游请求失败激增", Threshold: 1, CooldownSeconds: 60, Enabled: false},
		},
	}
	svc := newTestService(st, true)
	svc.prime()

	// Counter moves while the rule is disabled.
	metrics.Get().UpstreamFailures.Add(5)
	svc.Evaluate(context.Background())
	if len(st.claimed) != 0 {
		t.Fatalf("disabled rule must not fire, got %v", st.claimed)
	}

	// Re-enable: the backlog accumulated while disabled must not fire.
	st.rules[0].Enabled = true
	svc.Evaluate(context.Background())
	if len(st.claimed) != 0 {
		t.Fatalf("re-enabled rule must not fire on old backlog, got %v", st.claimed)
	}
}
