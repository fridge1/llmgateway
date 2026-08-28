// Package retention runs background user-retention jobs: a weekly usage report
// and layered silent-user winback notifications. Both are best-effort and
// deduplicated so a user never receives the same nudge twice in one period.
package retention

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Service owns the retention background goroutines.
type Service struct {
	store     store.Store
	cfgHolder *config.Holder

	cancel context.CancelFunc
	done   chan struct{}
}

// New creates a retention Service. Call Start to launch the background loop.
func New(s store.Store, cfgHolder *config.Holder) *Service {
	return &Service{store: s, cfgHolder: cfgHolder}
}

// Start launches the daily scheduler goroutine.
func (s *Service) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
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

// loop ticks hourly and dispatches jobs when their scheduled hour arrives.
// Hourly granularity keeps the implementation simple while ensuring each job
// fires once per day (dedup tables guard against double-sends within a day).
func (s *Service) loop(ctx context.Context) {
	defer close(s.done)
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tick(time.Now())
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) tick(now time.Time) {
	cfg := s.cfgHolder.Get().Retention

	// Weekly report: Monday 09:00.
	if cfg.WeeklyReportEnabled && now.Weekday() == time.Monday && now.Hour() == 9 {
		s.runWeeklyReport()
	}

	// Winback: every day at 10:00.
	if cfg.WinbackEnabled && now.Hour() == 10 {
		s.runWinback()
	}
}

// runWeeklyReport sends each recently-active user a summary of last week's usage.
func (s *Service) runWeeklyReport() {
	summaries, err := s.store.GetWeeklyUsageSummaries()
	if err != nil {
		slog.Error("retention: weekly summaries failed", "error", err)
		return
	}
	notifs := make([]store.Notification, 0, len(summaries))
	for _, w := range summaries {
		// Dedup per ISO week so a restart on Monday doesn't double-send.
		claimed, err := s.store.TryClaimNotificationDedup(w.UserID, "weekly_report")
		if err != nil || !claimed {
			continue
		}
		notifs = append(notifs, store.Notification{
			UserID:  w.UserID,
			Type:    "weekly_report",
			Title:   "上周用量周报",
			Content: weeklyReportContent(w),
		})
	}
	if err := s.store.BatchCreateNotifications(notifs); err != nil {
		slog.Error("retention: weekly report notifications failed", "error", err)
		return
	}
	if len(notifs) > 0 {
		slog.Info("retention: weekly report sent", "count", len(notifs))
	}
}

func weeklyReportContent(w store.WeeklyUsageSummary) string {
	return fmt.Sprintf(
		"上周你共发起 %d 次调用，处理约 %s token，消费 ¥%.2f。感谢你的使用，继续加油！",
		w.RequestCount, humanTokens(w.TotalTokens), w.TotalCost,
	)
}

// humanTokens formats large token counts compactly (e.g. 1.2万).
func humanTokens(n int64) string {
	if n >= 10000 {
		return fmt.Sprintf("%.1f万", float64(n)/10000)
	}
	return fmt.Sprintf("%d", n)
}

// winbackTier defines a silent-user segment and its message.
type winbackTier struct {
	kind    string
	minDays int
	maxDays int
	title   string
	content string
}

var winbackTiers = []winbackTier{
	{"winback_7d", 7, 14, "好久不见", "已经一周没见到你了，平台新增了不少模型与工具，回来看看吧。"},
	{"winback_14d", 14, 30, "我们想你了", "你已两周未使用，记得回来体验最新的 AI 编码能力。"},
	{"winback_30d", 30, 90, "专属回归礼", "超过一个月没来啦，回来即可领取专属体验金，期待你的回归。"},
}

// runWinback notifies silent users in layered tiers. Tiers are processed from
// longest-inactive to shortest so a user falls into exactly one bucket per run.
func (s *Service) runWinback() {
	total := 0
	for _, tier := range winbackTiers {
		ids, err := s.store.GetSilentUsers(tier.minDays, tier.maxDays)
		if err != nil {
			slog.Error("retention: silent users query failed", "tier", tier.kind, "error", err)
			continue
		}
		notifs := make([]store.Notification, 0, len(ids))
		for _, id := range ids {
			claimed, err := s.store.TryClaimNotificationDedup(id, tier.kind)
			if err != nil || !claimed {
				continue
			}
			notifs = append(notifs, store.Notification{
				UserID:  id,
				Type:    tier.kind,
				Title:   tier.title,
				Content: tier.content,
			})
		}
		if err := s.store.BatchCreateNotifications(notifs); err != nil {
			slog.Error("retention: winback notifications failed", "tier", tier.kind, "error", err)
			continue
		}
		total += len(notifs)
	}
	if total > 0 {
		slog.Info("retention: winback sent", "count", total)
	}
}
