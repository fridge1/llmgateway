package store

import (
	"testing"
	"time"
)

func TestRewardForStreak(t *testing.T) {
	base := 0.1
	tests := []struct {
		streak int
		want   float64
	}{
		{1, 0.1},  // day 1: base * 1
		{2, 0.2},  // day 2: base * 2
		{3, 0.3},  // day 3: base * 3
		{4, 0.5},  // day 4: base * 5
		{5, 0.7},  // day 5: base * 7
		{6, 1.0},  // day 6: base * 10
		{7, 2.0},  // day 7: base * 20
		{8, 0.1},  // day 8 wraps to day-1 multiplier
		{14, 2.0}, // day 14 wraps to day-7 multiplier
	}
	for _, tt := range tests {
		got := rewardForStreak(tt.streak, base)
		if got < tt.want-1e-9 || got > tt.want+1e-9 {
			t.Errorf("rewardForStreak(%d, %v) = %v, want %v", tt.streak, base, got, tt.want)
		}
	}
}

func TestRewardForStreakZeroBase(t *testing.T) {
	if got := rewardForStreak(5, 0); got != 0 {
		t.Errorf("rewardForStreak with zero base = %v, want 0", got)
	}
}

func TestDaysBetweenUTC(t *testing.T) {
	tests := []struct {
		earlier, later time.Time
		want           int
	}{
		{time.Date(2026, 6, 1, 23, 0, 0, 0, time.UTC), time.Date(2026, 6, 2, 1, 0, 0, 0, time.UTC), 1},
		{time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 1, 23, 0, 0, 0, time.UTC), 0},
		{time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC), 3},
	}
	for _, tt := range tests {
		if got := daysBetweenUTC(tt.earlier, tt.later); got != tt.want {
			t.Errorf("daysBetweenUTC(%v, %v) = %d, want %d", tt.earlier, tt.later, got, tt.want)
		}
	}
}
