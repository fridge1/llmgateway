package circuit

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/zhulang/llm-gateway/internal/metrics"
)

// IsUpstreamFailure reports whether err indicates a genuine upstream failure
// that should count toward the circuit breaker threshold.
// Client-initiated cancellations and gateway timeouts are excluded.
func IsUpstreamFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = iota // normal operation
	StateOpen                  // failing, requests denied
	StateHalfOpen              // recovery probe
)

// String returns a human-readable state name.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}

// Breaker is a thread-safe circuit breaker.
type Breaker struct {
	mu                  sync.Mutex
	state               State
	failureCount        int
	failureThreshold    int
	recoveryTimeout     time.Duration
	halfOpenMaxRequests int
	halfOpenCount       int
	lastFailureTime     time.Time
}

// NewBreaker creates a new Breaker in the Closed state.
func NewBreaker(failureThreshold int, recoveryTimeout time.Duration, halfOpenMaxRequests int) *Breaker {
	return &Breaker{
		state:               StateClosed,
		failureThreshold:    failureThreshold,
		recoveryTimeout:     recoveryTimeout,
		halfOpenMaxRequests: halfOpenMaxRequests,
	}
}

// State returns the current state of the breaker.
// If the breaker is Open and the recovery timeout has elapsed, it transitions to HalfOpen.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.currentState()
}

// currentState must be called with b.mu held.
func (b *Breaker) currentState() State {
	if b.state == StateOpen && time.Since(b.lastFailureTime) >= b.recoveryTimeout {
		b.transitionTo(StateHalfOpen)
	}
	return b.state
}

// transitionTo changes state and logs the transition. Must be called with b.mu held.
func (b *Breaker) transitionTo(newState State) {
	if b.state == newState {
		return
	}
	slog.Warn("circuit breaker state transition",
		"from", b.state.String(),
		"to", newState.String(),
	)
	if newState == StateOpen {
		metrics.Get().CircuitOpened.Add(1)
	}
	b.state = newState
	if newState == StateHalfOpen {
		b.halfOpenCount = 0
	}
}

// AllowRequest reports whether a request should be forwarded to the upstream.
// Closed: always true.
// Open: false unless recovery timeout elapsed, in which case transitions to HalfOpen and returns true.
// HalfOpen: true up to halfOpenMaxRequests, then false.
func (b *Breaker) AllowRequest() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := b.currentState()
	switch state {
	case StateClosed:
		return true
	case StateOpen:
		return false
	case StateHalfOpen:
		if b.halfOpenCount < b.halfOpenMaxRequests {
			b.halfOpenCount++
			return true
		}
		return false
	}
	return false
}

// RecordSuccess records a successful response.
// HalfOpen: transitions to Closed.
// Closed: resets failure count.
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateHalfOpen:
		b.failureCount = 0
		b.transitionTo(StateClosed)
	case StateClosed:
		b.failureCount = 0
	}
}

// RecordFailure records a failed response.
// HalfOpen: transitions back to Open.
// Closed: increments failure count, opens if threshold reached.
func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateHalfOpen:
		b.lastFailureTime = time.Now()
		b.transitionTo(StateOpen)
	case StateClosed:
		b.failureCount++
		if b.failureCount >= b.failureThreshold {
			b.lastFailureTime = time.Now()
			b.transitionTo(StateOpen)
		}
	}
}

// Reset forces the breaker back to Closed state, clearing failure count.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failureCount = 0
	b.transitionTo(StateClosed)
}

// Counts returns the current failure count and state for status reporting.
func (b *Breaker) Counts() (int, State) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.failureCount, b.currentState()
}
