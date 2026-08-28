// Package money provides helpers for CNY amount arithmetic and comparison.
//
// Balances are stored as DECIMAL(12,4) in PostgreSQL but handled as float64
// in Go, so direct comparisons like `available < price` fail probabilistically
// due to float noise and half-cent residues left by LLM billing. All amount
// comparisons on the subscription purchase path must go through this package.
package money

import "math"

// EpsilonCNY is the tolerance for amount comparisons.
// It covers both float64 noise (~1e-13) and the worst-case half-cent gap
// (0.0049) left by legacy orders whose payment amount was rounded down
// by %.2f before Ceil2 was introduced.
const EpsilonCNY = 0.005

// Ceil2 rounds v up to 2 decimal places (cent granularity).
// The 1e-6 guard prevents float noise from causing a spurious carry,
// e.g. 100.0-96.54 = 3.4600000000000084 must yield 3.46, not 3.47.
func Ceil2(v float64) float64 {
	return math.Ceil(v*100-1e-6) / 100
}

// Round2 rounds v to the nearest 2 decimal places.
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// GTE reports whether a >= b within EpsilonCNY tolerance.
func GTE(a, b float64) bool {
	return a-b > -EpsilonCNY
}
