package metrics

import "sync/atomic"

// Global metric counters exported for use across the application.

var (
	// BlockedIPRequestsTotal counts requests blocked by IP blacklist.
	BlockedIPRequestsTotal atomic.Int64
)

// IncrementBlockedIPRequests increments the blocked IP request counter.
func IncrementBlockedIPRequests() {
	BlockedIPRequestsTotal.Add(1)
	Get().BlockedIPHits.Add(1)
}

// IncrementRateLimitHitIP increments the IP rate limit counter.
func IncrementRateLimitHitIP() {
	Get().RateLimitHitsIP.Add(1)
	Get().RateLimitHitsTotal.Add(1)
}

// IncrementRateLimitHitUser increments the user rate limit counter.
func IncrementRateLimitHitUser() {
	Get().RateLimitHitsUser.Add(1)
	Get().RateLimitHitsTotal.Add(1)
}

// IncrementRateLimitHitAPIKey increments the API key rate limit counter.
func IncrementRateLimitHitAPIKey() {
	Get().RateLimitHitsAPIKey.Add(1)
	Get().RateLimitHitsTotal.Add(1)
}

// SetBlockedIPsTotal sets the current total number of blocked IPs.
func SetBlockedIPsTotal(count int) {
	Get().BlockedIPsTotal.Store(int64(count))
}

// IncrementBlockedIPsExpired increments the expired IP cleanup counter.
func IncrementBlockedIPsExpired(count int) {
	Get().BlockedIPsExpired.Add(int64(count))
}
