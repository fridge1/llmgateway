package subscription

import "testing"

func TestComputeShortfall(t *testing.T) {
	cases := []struct {
		name          string
		price         float64
		available     float64
		wantShortfall float64
		wantSuffic    bool
	}{
		{"exact balance", 100, 100, 0, true},
		{"balance above price", 100, 150, 0, true},
		{"within tolerance 0.0032 short", 100, 99.9968, 0, true},
		{"within tolerance 0.0049 short", 100, 99.9951, 0, true},
		{"just beyond tolerance floors at min order", 100, 99.992, 0.01, false},
		{"dead-zone gap 0.008 floors at min order", 100, 99.99 + 0.002, 0.01, false},
		{"four-decimal balance rounds up", 100, 96.5468, 3.46, false},
		{"two-decimal gap kept exact", 100, 96.54, 3.46, false},
		{"zero balance", 99.99, 0, 99.99, false},
		{"negative available clamped", 100, -5, 100, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, suffic := computeShortfall(c.price, c.available)
			if got != c.wantShortfall || suffic != c.wantSuffic {
				t.Errorf("computeShortfall(%v, %v) = (%v, %v), want (%v, %v)",
					c.price, c.available, got, suffic, c.wantShortfall, c.wantSuffic)
			}
		})
	}
}
