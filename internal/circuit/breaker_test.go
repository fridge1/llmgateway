package circuit

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestBreaker_StartsInClosedState(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	if b.State() != StateClosed {
		t.Errorf("expected StateClosed, got %s", b.State())
	}
}

func TestBreaker_OpensAfterThreshold(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	b.RecordFailure()
	b.RecordFailure()
	if b.State() != StateClosed {
		t.Errorf("expected StateClosed after 2 failures, got %s", b.State())
	}
	b.RecordFailure()
	if b.State() != StateOpen {
		t.Errorf("expected StateOpen after 3 failures, got %s", b.State())
	}
}

func TestBreaker_AllowRequest_ClosedAlwaysAllows(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	for i := 0; i < 10; i++ {
		if !b.AllowRequest() {
			t.Errorf("closed breaker should always allow requests")
		}
	}
}

func TestBreaker_AllowRequest_OpenDenies(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	b.RecordFailure()
	b.RecordFailure()
	b.RecordFailure()
	if b.State() != StateOpen {
		t.Fatalf("expected StateOpen, got %s", b.State())
	}
	if b.AllowRequest() {
		t.Errorf("open breaker should deny requests before recovery timeout")
	}
}

func TestBreaker_TransitionsToHalfOpen(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	b.RecordFailure()
	b.RecordFailure()
	b.RecordFailure()
	if b.State() != StateOpen {
		t.Fatalf("expected StateOpen, got %s", b.State())
	}
	time.Sleep(60 * time.Millisecond)
	if b.State() != StateHalfOpen {
		t.Errorf("expected StateHalfOpen after recovery timeout, got %s", b.State())
	}
}

func TestBreaker_HalfOpen_SuccessCloses(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	b.RecordFailure()
	b.RecordFailure()
	b.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	// Trigger transition to HalfOpen
	b.AllowRequest()
	if b.State() != StateHalfOpen {
		t.Fatalf("expected StateHalfOpen, got %s", b.State())
	}
	b.RecordSuccess()
	if b.State() != StateClosed {
		t.Errorf("expected StateClosed after success in HalfOpen, got %s", b.State())
	}
}

func TestBreaker_HalfOpen_FailureReopens(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	b.RecordFailure()
	b.RecordFailure()
	b.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	b.AllowRequest()
	if b.State() != StateHalfOpen {
		t.Fatalf("expected StateHalfOpen, got %s", b.State())
	}
	b.RecordFailure()
	if b.State() != StateOpen {
		t.Errorf("expected StateOpen after failure in HalfOpen, got %s", b.State())
	}
}

func TestBreaker_SuccessResetsFailureCount(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	b.RecordFailure()
	b.RecordFailure()
	if b.State() != StateClosed {
		t.Fatalf("expected StateClosed after 2 failures, got %s", b.State())
	}
	b.RecordSuccess()
	// failure count should be reset; 2 more failures should not open it
	b.RecordFailure()
	b.RecordFailure()
	if b.State() != StateClosed {
		t.Errorf("expected StateClosed (failure count was reset), got %s", b.State())
	}
}

func TestBreaker_Reset_FromOpen(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	b.RecordFailure()
	b.RecordFailure()
	b.RecordFailure()
	if b.State() != StateOpen {
		t.Fatalf("expected StateOpen, got %s", b.State())
	}
	b.Reset()
	if b.State() != StateClosed {
		t.Errorf("expected StateClosed after Reset, got %s", b.State())
	}
	// Verify failure count was cleared
	b.RecordFailure()
	b.RecordFailure()
	if b.State() != StateClosed {
		t.Errorf("expected StateClosed (failure count was reset), got %s", b.State())
	}
}

func TestBreaker_Reset_FromHalfOpen(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	b.RecordFailure()
	b.RecordFailure()
	b.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	b.AllowRequest()
	if b.State() != StateHalfOpen {
		t.Fatalf("expected StateHalfOpen, got %s", b.State())
	}
	b.Reset()
	if b.State() != StateClosed {
		t.Errorf("expected StateClosed after Reset from HalfOpen, got %s", b.State())
	}
}

func TestBreaker_Reset_ClearsPartialFailures(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	b.RecordFailure()
	b.RecordFailure()
	b.Reset()
	// failure count cleared; 2 more failures should not open
	b.RecordFailure()
	b.RecordFailure()
	if b.State() != StateClosed {
		t.Errorf("expected StateClosed after Reset cleared partial failures, got %s", b.State())
	}
}

func TestBreaker_HalfOpen_LimitsRequests(t *testing.T) {
	b := NewBreaker(3, 50*time.Millisecond, 2)
	b.RecordFailure()
	b.RecordFailure()
	b.RecordFailure()
	time.Sleep(60 * time.Millisecond)

	// First request should trigger HalfOpen and be allowed
	if !b.AllowRequest() {
		t.Errorf("first request in HalfOpen should be allowed")
	}
	// Second request should be allowed (halfOpenMaxRequests=2)
	if !b.AllowRequest() {
		t.Errorf("second request in HalfOpen should be allowed")
	}
	// Third request should be denied
	if b.AllowRequest() {
		t.Errorf("third request in HalfOpen should be denied (halfOpenMaxRequests=2)")
	}
}

func TestIsUpstreamFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"context canceled", context.Canceled, false},
		{"wrapped context canceled", fmt.Errorf("Post \"https://example.com\": %w", context.Canceled), false},
		{"deadline exceeded", context.DeadlineExceeded, false},
		{"wrapped deadline exceeded", fmt.Errorf("Post \"https://example.com\": %w", context.DeadlineExceeded), false},
		{"connection refused", errors.New("dial tcp: connection refused"), true},
		{"generic error", errors.New("something broke"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUpstreamFailure(tt.err); got != tt.want {
				t.Errorf("IsUpstreamFailure(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
