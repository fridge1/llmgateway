package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds application-wide metrics using atomic counters.
// This is a lightweight implementation using the standard library.
// Can be replaced with prometheus/client_golang for production Grafana integration.
type Metrics struct {
	// Request counters
	RequestsTotal   atomic.Int64
	RequestsErrors  atomic.Int64
	RequestsStream  atomic.Int64

	// Billing counters
	BillingJobsTotal    atomic.Int64
	BillingJobsDropped  atomic.Int64
	BillingQueueOverflow atomic.Int64

	// Upstream counters
	UpstreamRequests  atomic.Int64
	UpstreamFailures  atomic.Int64
	UpstreamFailovers atomic.Int64

	// Retry metrics
	UpstreamRetries          atomic.Int64 // Total retry attempts
	UpstreamRetriesSucceeded atomic.Int64 // Successful retries

	// Circuit breaker counters
	CircuitOpened atomic.Int64

	// Auth counters
	AuthCacheHits   atomic.Int64
	AuthCacheMisses atomic.Int64

	// Latency tracking (simplified histogram using buckets)
	latencyMu      sync.Mutex
	latencyBuckets [8]atomic.Int64 // <10ms, <50ms, <100ms, <500ms, <1s, <5s, <30s, >30s
	latencyTotal   atomic.Int64
	latencySum     atomic.Int64 // in milliseconds

	// Gauge values
	ActiveConnections atomic.Int64
	BillingQueueDepth atomic.Int64
	GoroutineCount    atomic.Int64

	// Rate limiting counters
	RateLimitHitsIP     atomic.Int64 // IP 层限流触发次数
	RateLimitHitsUser   atomic.Int64 // 用户层限流触发次数
	RateLimitHitsAPIKey atomic.Int64 // API Key 层限流触发次数
	RateLimitHitsTotal  atomic.Int64 // 总限流触发次数

	// IP blocking counters
	BlockedIPHits     atomic.Int64 // 黑名单 IP 拦截次数
	BlockedIPsTotal   atomic.Int64 // 当前黑名单 IP 总数（gauge）
	BlockedIPsExpired atomic.Int64 // 过期自动清理的 IP 总数
}

// Global metrics instance
var global = &Metrics{}

// Get returns the global metrics instance.
func Get() *Metrics { return global }

// RecordLatency records a request latency measurement.
func (m *Metrics) RecordLatency(d time.Duration) {
	ms := d.Milliseconds()
	m.latencyTotal.Add(1)
	m.latencySum.Add(ms)

	switch {
	case ms < 10:
		m.latencyBuckets[0].Add(1)
	case ms < 50:
		m.latencyBuckets[1].Add(1)
	case ms < 100:
		m.latencyBuckets[2].Add(1)
	case ms < 500:
		m.latencyBuckets[3].Add(1)
	case ms < 1000:
		m.latencyBuckets[4].Add(1)
	case ms < 5000:
		m.latencyBuckets[5].Add(1)
	case ms < 30000:
		m.latencyBuckets[6].Add(1)
	default:
		m.latencyBuckets[7].Add(1)
	}
}
