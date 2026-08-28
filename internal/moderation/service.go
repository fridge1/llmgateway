// Package moderation screens user prompts against a keyword rule set before
// requests are forwarded upstream. Rules and switches live in the DB and are
// cached in memory with periodic refresh, so the per-request check is a pure
// in-memory scan with no allocation on the hot path.
package moderation

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/zhulang/llm-gateway/internal/store"
)

// Verdict is the result of a moderation check.
type Verdict struct {
	Flagged     bool
	MatchedRule string
	Snippet     string
}

// Moderator checks a piece of text against content rules.
type Moderator interface {
	Check(text string) Verdict
}

// snapshot is the immutable cached rule state swapped atomically on refresh.
type snapshot struct {
	enabled     bool
	enforceAll  bool
	keywords    []string        // lowercased, enabled keywords
	modelSet    map[string]bool // models with moderation_enabled (when !enforceAll)
	tenantSet   map[string]bool // tenants with moderation_enabled (when !enforceAll)
	lowerJoined string          // unused placeholder for future Aho-Corasick swap
}

// Service is the DB-backed keyword moderator with an in-memory cache.
type Service struct {
	store store.Store

	mu   sync.RWMutex
	snap *snapshot

	cancel chan struct{}
	done   chan struct{}
}

// NewService creates the moderation service and loads the initial rule set.
func NewService(s store.Store) *Service {
	svc := &Service{store: s, snap: &snapshot{}}
	svc.Refresh()
	return svc
}

// Start launches the periodic cache refresh loop (every 30s).
func (s *Service) Start() {
	s.cancel = make(chan struct{})
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Refresh()
			case <-s.cancel:
				return
			}
		}
	}()
}

// Stop terminates the refresh loop.
func (s *Service) Stop() {
	if s.cancel == nil {
		return
	}
	close(s.cancel)
	<-s.done
}

// Refresh reloads settings, keywords and opt-in targets from the DB.
func (s *Service) Refresh() {
	settings, err := s.store.GetModerationSettings()
	if err != nil {
		slog.Error("moderation: load settings failed", "error", err)
		return
	}
	next := &snapshot{enabled: settings.Enabled, enforceAll: settings.EnforceAll}

	if settings.Enabled {
		kws, err := s.store.ListModerationKeywords()
		if err != nil {
			slog.Error("moderation: load keywords failed", "error", err)
			return
		}
		for _, k := range kws {
			if k.Enabled && k.Keyword != "" {
				next.keywords = append(next.keywords, strings.ToLower(k.Keyword))
			}
		}
		if !settings.EnforceAll {
			models, tenants, err := s.store.ListModerationEnabledTargets()
			if err != nil {
				slog.Error("moderation: load targets failed", "error", err)
				return
			}
			next.modelSet = make(map[string]bool, len(models))
			for _, m := range models {
				next.modelSet[m] = true
			}
			next.tenantSet = make(map[string]bool, len(tenants))
			for _, t := range tenants {
				next.tenantSet[t] = true
			}
		}
	}

	s.mu.Lock()
	s.snap = next
	s.mu.Unlock()
}

func (s *Service) snapshot() *snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

// Applicable reports whether moderation applies to this model/tenant combo.
// tenantID is empty for personal-account requests.
func (s *Service) Applicable(model, tenantID string) bool {
	snap := s.snapshot()
	if !snap.enabled || len(snap.keywords) == 0 {
		return false
	}
	if snap.enforceAll {
		return true
	}
	if snap.modelSet[model] {
		return true
	}
	if tenantID != "" && snap.tenantSet[tenantID] {
		return true
	}
	return false
}

// Check scans text (case-insensitive) against the cached keyword set.
func (s *Service) Check(text string) Verdict {
	snap := s.snapshot()
	if len(snap.keywords) == 0 || text == "" {
		return Verdict{}
	}
	lower := strings.ToLower(text)
	for _, kw := range snap.keywords {
		if idx := strings.Index(lower, kw); idx >= 0 {
			return Verdict{
				Flagged:     true,
				MatchedRule: kw,
				Snippet:     snippetAround(text, idx, len(kw)),
			}
		}
	}
	return Verdict{}
}

// CheckAndRecord runs Check and persists a hit record when flagged.
// userID/tenantID may be empty; they are stored as NULL then.
func (s *Service) CheckAndRecord(text, model, userID, tenantID string) Verdict {
	v := s.Check(text)
	if !v.Flagged {
		return v
	}
	var uidPtr, tidPtr *string
	if userID != "" {
		uidPtr = &userID
	}
	if tenantID != "" {
		tidPtr = &tenantID
	}
	if err := s.store.CreateModerationHit(uidPtr, tidPtr, model, v.MatchedRule, v.Snippet); err != nil {
		slog.Error("moderation: record hit failed", "error", err)
	}
	return v
}

// snippetAround returns up to 60 runes of context around the match, keeping
// the stored evidence small and avoiding logging entire prompts.
func snippetAround(text string, idx, matchLen int) string {
	start := idx - 20
	if start < 0 {
		start = 0
	}
	end := idx + matchLen + 20
	if end > len(text) {
		end = len(text)
	}
	// Align to valid UTF-8 boundaries.
	for start > 0 && start < len(text) && !utf8Start(text[start]) {
		start--
	}
	for end < len(text) && !utf8Start(text[end]) {
		end++
	}
	snip := text[start:end]
	if len(snip) > 180 {
		snip = snip[:180]
	}
	return snip
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }
