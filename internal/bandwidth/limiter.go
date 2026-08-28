package bandwidth

import (
	"context"
	"fmt"
	"time"
)

// Limiter controls concurrent image response writes to avoid bandwidth saturation.
type Limiter struct {
	sem     chan struct{}
	timeout time.Duration
}

// NewLimiter creates a bandwidth limiter.
// maxConcurrent: max simultaneous image response writes.
// timeout: max time to wait for a slot before giving up.
func NewLimiter(maxConcurrent int, timeout time.Duration) *Limiter {
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	return &Limiter{
		sem:     make(chan struct{}, maxConcurrent),
		timeout: timeout,
	}
}

// Acquire blocks until a slot is available, ctx is cancelled, or timeout expires.
func (l *Limiter) Acquire(ctx context.Context) (release func(), err error) {
	timer := time.NewTimer(l.timeout)
	defer timer.Stop()

	select {
	case l.sem <- struct{}{}:
		return func() { <-l.sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, fmt.Errorf("bandwidth limiter: timed out after %v", l.timeout)
	}
}
