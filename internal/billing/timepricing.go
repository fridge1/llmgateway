package billing

import (
	"time"

	"github.com/zhulang/llm-gateway/internal/store"
)

var shanghaiLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// Fallback: UTC+8 fixed offset if timezone data unavailable.
		loc = time.FixedZone("CST", 8*3600)
	}
	return loc
}()

// resolveMultiplier returns the price multiplier for a given UTC time
// according to the model's time_based_rules.
// Rules are evaluated in order; the first match wins.
// Returns 1.0 if no rule matches (base price, no change).
func resolveMultiplier(rules []store.TimeBasedPricingRule, t time.Time) float64 {
	if len(rules) == 0 {
		return 1.0
	}
	local := t.In(shanghaiLoc)
	weekday := int(local.Weekday()) // 0=Sunday
	hhmm := local.Hour()*60 + local.Minute()

	for _, r := range rules {
		if !containsDay(r.Days, weekday) {
			continue
		}
		start := parseHHMM(r.StartTime)
		end := parseHHMM(r.EndTime)
		if inTimeWindow(hhmm, start, end) {
			if r.Multiplier <= 0 {
				return 1.0
			}
			return r.Multiplier
		}
	}
	return 1.0
}

// applyTimeMultiplier returns a shallow-copied ModelPricing with all price
// fields scaled by multiplier. The original is never modified.
// PricingTiers are scaled per tier so tier-based billing also respects the window.
func applyTimeMultiplier(p *store.ModelPricing, multiplier float64) *store.ModelPricing {
	if multiplier == 1.0 {
		return p // fast path: no copy needed
	}
	cp := *p
	cp.InputPrice = p.InputPrice * multiplier
	cp.OutputPrice = p.OutputPrice * multiplier
	cp.CachedInputPrice = p.CachedInputPrice * multiplier
	cp.CacheCreationPrice = p.CacheCreationPrice * multiplier
	cp.CacheCreation1hPrice = p.CacheCreation1hPrice * multiplier
	if len(p.PricingTiers) > 0 {
		tiers := make([]store.PricingTier, len(p.PricingTiers))
		for i, t := range p.PricingTiers {
			tiers[i] = store.PricingTier{
				MinTokens:        t.MinTokens,
				MaxTokens:        t.MaxTokens,
				InputPrice:       t.InputPrice * multiplier,
				OutputPrice:      t.OutputPrice * multiplier,
				CachedInputPrice: t.CachedInputPrice * multiplier,
			}
		}
		cp.PricingTiers = tiers
	}
	return &cp
}

func containsDay(days []int, d int) bool {
	for _, v := range days {
		if v == d {
			return true
		}
	}
	return false
}

// parseHHMM converts "HH:MM" to minutes-since-midnight. Returns 0 on error.
func parseHHMM(s string) int {
	if len(s) != 5 || s[2] != ':' {
		return 0
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0
	}
	return h*60 + m
}

// inTimeWindow returns true if minuteOfDay is in [start, end).
// Supports overnight windows (end < start, e.g. 22:00–06:00).
func inTimeWindow(minuteOfDay, start, end int) bool {
	if start == end {
		return true // all-day rule
	}
	if start < end {
		return minuteOfDay >= start && minuteOfDay < end
	}
	// Overnight: e.g. start=22*60, end=6*60 → midnight crosses the window
	return minuteOfDay >= start || minuteOfDay < end
}
