package billing

import (
	"testing"
	"time"

	"github.com/zhulang/llm-gateway/internal/store"
)

func TestResolveMultiplier_NoRules(t *testing.T) {
	now := time.Now()
	multiplier := resolveMultiplier(nil, now)
	if multiplier != 1.0 {
		t.Errorf("expected 1.0, got %f", multiplier)
	}

	multiplier = resolveMultiplier([]store.TimeBasedPricingRule{}, now)
	if multiplier != 1.0 {
		t.Errorf("expected 1.0, got %f", multiplier)
	}
}

func TestResolveMultiplier_SingleRuleMatch(t *testing.T) {
	// 2024-01-15 Monday 10:00 CST
	loc := time.FixedZone("CST", 8*3600)
	testTime := time.Date(2024, 1, 15, 10, 0, 0, 0, loc)

	rules := []store.TimeBasedPricingRule{
		{
			Name:       "工作日高峰",
			Days:       []int{1, 2, 3, 4, 5}, // Mon-Fri
			StartTime:  "09:00",
			EndTime:    "18:00",
			Multiplier: 1.5,
		},
	}

	multiplier := resolveMultiplier(rules, testTime)
	if multiplier != 1.5 {
		t.Errorf("expected 1.5, got %f", multiplier)
	}
}

func TestResolveMultiplier_SingleRuleNoMatch(t *testing.T) {
	// 2024-01-14 Sunday 10:00 CST (weekend)
	loc := time.FixedZone("CST", 8*3600)
	testTime := time.Date(2024, 1, 14, 10, 0, 0, 0, loc)

	rules := []store.TimeBasedPricingRule{
		{
			Name:       "工作日高峰",
			Days:       []int{1, 2, 3, 4, 5}, // Mon-Fri only
			StartTime:  "09:00",
			EndTime:    "18:00",
			Multiplier: 1.5,
		},
	}

	multiplier := resolveMultiplier(rules, testTime)
	if multiplier != 1.0 {
		t.Errorf("expected 1.0 (no match), got %f", multiplier)
	}
}

func TestResolveMultiplier_OvernightWindow(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)

	rules := []store.TimeBasedPricingRule{
		{
			Name:       "深夜空闲",
			Days:       []int{0, 1, 2, 3, 4, 5, 6}, // all days
			StartTime:  "22:00",
			EndTime:    "06:00",
			Multiplier: 0.8,
		},
	}

	// 23:00 - should match (after start)
	t1 := time.Date(2024, 1, 15, 23, 0, 0, 0, loc)
	if m := resolveMultiplier(rules, t1); m != 0.8 {
		t.Errorf("23:00 should match overnight window, got %f", m)
	}

	// 03:00 - should match (before end)
	t2 := time.Date(2024, 1, 16, 3, 0, 0, 0, loc)
	if m := resolveMultiplier(rules, t2); m != 0.8 {
		t.Errorf("03:00 should match overnight window, got %f", m)
	}

	// 10:00 - should not match
	t3 := time.Date(2024, 1, 15, 10, 0, 0, 0, loc)
	if m := resolveMultiplier(rules, t3); m != 1.0 {
		t.Errorf("10:00 should not match overnight window, got %f", m)
	}
}

func TestResolveMultiplier_FirstMatchWins(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	testTime := time.Date(2024, 1, 15, 10, 0, 0, 0, loc) // Monday 10:00

	rules := []store.TimeBasedPricingRule{
		{
			Name:       "规则1",
			Days:       []int{1, 2, 3, 4, 5},
			StartTime:  "09:00",
			EndTime:    "18:00",
			Multiplier: 1.5,
		},
		{
			Name:       "规则2（也匹配但应被忽略）",
			Days:       []int{1, 2, 3, 4, 5},
			StartTime:  "08:00",
			EndTime:    "12:00",
			Multiplier: 2.0,
		},
	}

	multiplier := resolveMultiplier(rules, testTime)
	if multiplier != 1.5 {
		t.Errorf("expected first match (1.5), got %f", multiplier)
	}
}

func TestResolveMultiplier_WeekdayFilter(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)

	rules := []store.TimeBasedPricingRule{
		{
			Name:       "工作日",
			Days:       []int{1, 2, 3, 4, 5},
			StartTime:  "09:00",
			EndTime:    "18:00",
			Multiplier: 1.5,
		},
	}

	// Monday should match
	monday := time.Date(2024, 1, 15, 10, 0, 0, 0, loc)
	if m := resolveMultiplier(rules, monday); m != 1.5 {
		t.Errorf("Monday should match, got %f", m)
	}

	// Sunday should not match
	sunday := time.Date(2024, 1, 14, 10, 0, 0, 0, loc)
	if m := resolveMultiplier(rules, sunday); m != 1.0 {
		t.Errorf("Sunday should not match, got %f", m)
	}
}

func TestParseHHMM_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"00:00", 0},
		{"09:00", 540},
		{"12:30", 750},
		{"23:59", 1439},
	}

	for _, tt := range tests {
		result := parseHHMM(tt.input)
		if result != tt.expected {
			t.Errorf("parseHHMM(%s) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestParseHHMM_Invalid(t *testing.T) {
	invalid := []string{"", "9:00", "09:0", "09-00", "25:00", "abc"}
	for _, s := range invalid {
		result := parseHHMM(s)
		if result != 0 {
			t.Errorf("parseHHMM(%s) should return 0 for invalid input, got %d", s, result)
		}
	}
}

func TestApplyTimeMultiplier_ScalesAllFields(t *testing.T) {
	p := &store.ModelPricing{
		InputPrice:           10.0,
		OutputPrice:          20.0,
		CachedInputPrice:     2.0,
		CacheCreationPrice:   30.0,
		CacheCreation1hPrice: 40.0,
	}

	result := applyTimeMultiplier(p, 1.5)

	if result.InputPrice != 15.0 {
		t.Errorf("InputPrice: expected 15.0, got %f", result.InputPrice)
	}
	if result.OutputPrice != 30.0 {
		t.Errorf("OutputPrice: expected 30.0, got %f", result.OutputPrice)
	}
	if result.CachedInputPrice != 3.0 {
		t.Errorf("CachedInputPrice: expected 3.0, got %f", result.CachedInputPrice)
	}
	if result.CacheCreationPrice != 45.0 {
		t.Errorf("CacheCreationPrice: expected 45.0, got %f", result.CacheCreationPrice)
	}
	if result.CacheCreation1hPrice != 60.0 {
		t.Errorf("CacheCreation1hPrice: expected 60.0, got %f", result.CacheCreation1hPrice)
	}
}

func TestApplyTimeMultiplier_ScalesTiers(t *testing.T) {
	p := &store.ModelPricing{
		InputPrice:  10.0,
		OutputPrice: 20.0,
		PricingTiers: []store.PricingTier{
			{MinTokens: 0, MaxTokens: 1000, InputPrice: 5.0, OutputPrice: 10.0, CachedInputPrice: 1.0},
			{MinTokens: 1000, MaxTokens: 10000, InputPrice: 4.0, OutputPrice: 8.0, CachedInputPrice: 0.8},
		},
	}

	result := applyTimeMultiplier(p, 2.0)

	if len(result.PricingTiers) != 2 {
		t.Fatalf("expected 2 tiers, got %d", len(result.PricingTiers))
	}

	if result.PricingTiers[0].InputPrice != 10.0 {
		t.Errorf("tier[0] InputPrice: expected 10.0, got %f", result.PricingTiers[0].InputPrice)
	}
	if result.PricingTiers[1].OutputPrice != 16.0 {
		t.Errorf("tier[1] OutputPrice: expected 16.0, got %f", result.PricingTiers[1].OutputPrice)
	}
}

func TestApplyTimeMultiplier_Multiplier1_NoCopy(t *testing.T) {
	p := &store.ModelPricing{
		InputPrice:  10.0,
		OutputPrice: 20.0,
	}

	result := applyTimeMultiplier(p, 1.0)

	if result != p {
		t.Error("multiplier 1.0 should return original pointer (fast path)")
	}
}

func TestInTimeWindow_Normal(t *testing.T) {
	start := parseHHMM("09:00") // 540
	end := parseHHMM("18:00")   // 1080

	tests := []struct {
		minute   int
		expected bool
	}{
		{540, true},   // 09:00 - at start
		{900, true},   // 15:00 - in middle
		{1079, true},  // 17:59 - just before end
		{1080, false}, // 18:00 - at end (exclusive)
		{300, false},  // 05:00 - before start
		{1200, false}, // 20:00 - after end
	}

	for _, tt := range tests {
		result := inTimeWindow(tt.minute, start, end)
		if result != tt.expected {
			t.Errorf("inTimeWindow(%d, %d, %d) = %v, want %v", tt.minute, start, end, result, tt.expected)
		}
	}
}

func TestInTimeWindow_Overnight(t *testing.T) {
	start := parseHHMM("22:00") // 1320
	end := parseHHMM("06:00")   // 360

	tests := []struct {
		minute   int
		expected bool
	}{
		{1320, true}, // 22:00 - at start
		{1400, true}, // 23:20 - after start
		{0, true},    // 00:00 - midnight
		{300, true},  // 05:00 - before end
		{359, true},  // 05:59 - just before end
		{360, false}, // 06:00 - at end (exclusive)
		{720, false}, // 12:00 - middle of day (not in window)
	}

	for _, tt := range tests {
		result := inTimeWindow(tt.minute, start, end)
		if result != tt.expected {
			t.Errorf("inTimeWindow(%d, %d, %d) = %v, want %v", tt.minute, start, end, result, tt.expected)
		}
	}
}

func TestInTimeWindow_AllDay(t *testing.T) {
	start := parseHHMM("00:00")
	end := parseHHMM("00:00")

	// When start == end, it's an all-day rule
	if !inTimeWindow(720, start, end) {
		t.Error("all-day rule (start==end) should match any time")
	}
}
