package metrics

import (
	"encoding/json"
	"net/http"
	"runtime"
)

// Handler returns an HTTP handler that exposes metrics as JSON.
// Path: GET /metrics
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := Get()

		// Update goroutine count
		m.GoroutineCount.Store(int64(runtime.NumGoroutine()))

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		snapshot := map[string]any{
			"requests": map[string]int64{
				"total":  m.RequestsTotal.Load(),
				"errors": m.RequestsErrors.Load(),
				"stream": m.RequestsStream.Load(),
			},
			"billing": map[string]int64{
				"jobs_total":     m.BillingJobsTotal.Load(),
				"jobs_dropped":   m.BillingJobsDropped.Load(),
				"queue_overflow": m.BillingQueueOverflow.Load(),
				"queue_depth":    m.BillingQueueDepth.Load(),
			},
			"upstream": map[string]int64{
				"requests":  m.UpstreamRequests.Load(),
				"failures":  m.UpstreamFailures.Load(),
				"failovers": m.UpstreamFailovers.Load(),
			},
			"circuit": map[string]int64{
				"opened": m.CircuitOpened.Load(),
			},
			"auth_cache": map[string]int64{
				"hits":   m.AuthCacheHits.Load(),
				"misses": m.AuthCacheMisses.Load(),
			},
			"rate_limit": map[string]int64{
				"hits_ip":      m.RateLimitHitsIP.Load(),
				"hits_user":    m.RateLimitHitsUser.Load(),
				"hits_api_key": m.RateLimitHitsAPIKey.Load(),
				"hits_total":   m.RateLimitHitsTotal.Load(),
			},
			"ip_blocking": map[string]int64{
				"blocked_requests": m.BlockedIPHits.Load(),
				"blocked_ips":      m.BlockedIPsTotal.Load(),
				"expired_cleanups": m.BlockedIPsExpired.Load(),
			},
			"latency": map[string]any{
				"total_requests": m.latencyTotal.Load(),
				"sum_ms":         m.latencySum.Load(),
				"buckets": map[string]int64{
					"lt_10ms":  m.latencyBuckets[0].Load(),
					"lt_50ms":  m.latencyBuckets[1].Load(),
					"lt_100ms": m.latencyBuckets[2].Load(),
					"lt_500ms": m.latencyBuckets[3].Load(),
					"lt_1s":    m.latencyBuckets[4].Load(),
					"lt_5s":    m.latencyBuckets[5].Load(),
					"lt_30s":   m.latencyBuckets[6].Load(),
					"gt_30s":   m.latencyBuckets[7].Load(),
				},
			},
			"runtime": map[string]any{
				"goroutines":     m.GoroutineCount.Load(),
				"active_conns":   m.ActiveConnections.Load(),
				"heap_alloc_mb":  memStats.HeapAlloc / 1024 / 1024,
				"heap_sys_mb":    memStats.HeapSys / 1024 / 1024,
				"gc_cycles":      memStats.NumGC,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snapshot)
	}
}
