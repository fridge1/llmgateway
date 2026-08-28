package balancer

import (
	"errors"
	"sync/atomic"

	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/config"
)

var ErrNoAvailableUpstream = errors.New("all upstreams unavailable")

// Upstream pairs config with its circuit breaker.
type Upstream struct {
	Config  config.UpstreamConfig
	Breaker *circuit.Breaker
}

// RoundRobin selects upstreams in order, skipping those with open breakers.
type RoundRobin struct {
	counter atomic.Uint64
}

// NewRoundRobin creates a new RoundRobin load balancer.
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

// Next returns the next available upstream, skipping those with open circuit
// breakers. It increments the internal counter atomically and tries each
// upstream in order starting from the counter position. Returns
// ErrNoAvailableUpstream if all upstreams are unavailable.
func (rr *RoundRobin) Next(upstreams []Upstream) (*Upstream, error) {
	n := uint64(len(upstreams))
	if n == 0 {
		return nil, ErrNoAvailableUpstream
	}

	start := rr.counter.Add(1) - 1

	for i := uint64(0); i < n; i++ {
		idx := (start + i) % n
		u := &upstreams[idx]
		if u.Breaker.AllowRequest() {
			return u, nil
		}
	}

	return nil, ErrNoAvailableUpstream
}

// Counter returns the current counter value. It increments atomically and is
// used by the proxy handler for failover starting index.
func (rr *RoundRobin) Counter() uint64 {
	return rr.counter.Add(1)
}
