package money

import "testing"

func TestCeil2(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want float64
	}{
		{"four-decimal shortfall rounds up", 3.4532, 3.46},
		{"float noise does not carry", 100.0 - 96.54, 3.46},
		{"sub-cent rounds up to one cent", 0.0049, 0.01},
		{"exact cent unchanged", 0.01, 0.01},
		{"exact two decimals unchanged", 3.46, 3.46},
		{"zero", 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Ceil2(c.in); got != c.want {
				t.Errorf("Ceil2(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestRound2(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{3.4549, 3.45},
		{3.455, 3.46},
		{0.005, 0.01},
	}
	for _, c := range cases {
		if got := Round2(c.in); got != c.want {
			t.Errorf("Round2(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestGTE(t *testing.T) {
	cases := []struct {
		name string
		a, b float64
		want bool
	}{
		{"equal", 100, 100, true},
		{"within tolerance below", 99.9951, 100, true},
		{"beyond tolerance below", 99.9949, 100, false},
		{"float noise below", 100 - 1e-13, 100, true},
		{"clearly above", 100.01, 100, true},
		{"clearly below", 96.5468, 100, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GTE(c.a, c.b); got != c.want {
				t.Errorf("GTE(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}
