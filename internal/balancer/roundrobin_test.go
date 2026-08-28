package balancer

import (
	"testing"
	"time"

	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/config"
)

// makeUpstreams creates upstreams with breakers (threshold=3, timeout=10s, halfOpen=1).
func makeUpstreams(names ...string) []Upstream {
	upstreams := make([]Upstream, len(names))
	for i, name := range names {
		upstreams[i] = Upstream{
			Config: config.UpstreamConfig{
				Provider: name,
				BaseURL:  "http://" + name + ".example.com",
			},
			Breaker: circuit.NewBreaker(3, 10*time.Second, 1),
		}
	}
	return upstreams
}

// tripBreaker trips the breaker by recording failures up to the threshold.
func tripBreaker(b *circuit.Breaker, threshold int) {
	for i := 0; i < threshold; i++ {
		b.RecordFailure()
	}
}

func TestRoundRobin_CyclesThroughUpstreams(t *testing.T) {
	rr := NewRoundRobin()
	upstreams := makeUpstreams("a", "b", "c")

	counts := make(map[string]int)
	for i := 0; i < 6; i++ {
		u, err := rr.Next(upstreams)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		counts[u.Config.Provider]++
	}

	for _, name := range []string{"a", "b", "c"} {
		if counts[name] != 2 {
			t.Errorf("upstream %q selected %d times, want 2", name, counts[name])
		}
	}
}

func TestRoundRobin_SkipsOpenBreaker(t *testing.T) {
	rr := NewRoundRobin()
	upstreams := makeUpstreams("a", "b")

	// Trip breaker on "a"
	tripBreaker(upstreams[0].Breaker, 3)

	for i := 0; i < 6; i++ {
		u, err := rr.Next(upstreams)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if u.Config.Provider != "b" {
			t.Errorf("call %d: got upstream %q, want %q", i, u.Config.Provider, "b")
		}
	}
}

func TestRoundRobin_AllBreakersOpen_ReturnsError(t *testing.T) {
	rr := NewRoundRobin()
	upstreams := makeUpstreams("a", "b", "c")

	// Trip all breakers
	for i := range upstreams {
		tripBreaker(upstreams[i].Breaker, 3)
	}

	_, err := rr.Next(upstreams)
	if err != ErrNoAvailableUpstream {
		t.Errorf("got error %v, want ErrNoAvailableUpstream", err)
	}
}

func TestRoundRobin_SingleUpstream(t *testing.T) {
	rr := NewRoundRobin()
	upstreams := makeUpstreams("only")

	for i := 0; i < 3; i++ {
		u, err := rr.Next(upstreams)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if u.Config.Provider != "only" {
			t.Errorf("call %d: got upstream %q, want %q", i, u.Config.Provider, "only")
		}
	}
}
