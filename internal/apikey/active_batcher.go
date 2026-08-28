package apikey

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// ActiveStore is the subset of store.Store needed by ActiveBatcher.
type ActiveStore interface {
	BatchTouchUsersLastActive(ids []string) error
}

// ActiveBatcher batches user last_active_at updates into periodic bulk writes.
// It mirrors TouchBatcher but tracks user IDs (for retention/DAU stats) rather
// than API key IDs. DB-side throttling (only writing when stale) keeps the
// write volume low even under high request rates.
type ActiveBatcher struct {
	store    ActiveStore
	interval time.Duration

	mu  sync.Mutex
	ids map[string]struct{}

	cancel context.CancelFunc
	done   chan struct{}
}

// NewActiveBatcher creates a batcher that flushes every interval.
func NewActiveBatcher(store ActiveStore, interval time.Duration) *ActiveBatcher {
	ctx, cancel := context.WithCancel(context.Background())
	b := &ActiveBatcher{
		store:    store,
		interval: interval,
		ids:      make(map[string]struct{}),
		cancel:   cancel,
		done:     make(chan struct{}),
	}
	go b.loop(ctx)
	return b
}

// Touch enqueues a user ID for a batched last_active_at update.
func (b *ActiveBatcher) Touch(id string) {
	if id == "" {
		return
	}
	b.mu.Lock()
	b.ids[id] = struct{}{}
	b.mu.Unlock()
}

// Stop flushes remaining IDs and stops the background goroutine.
func (b *ActiveBatcher) Stop() {
	b.cancel()
	<-b.done
}

func (b *ActiveBatcher) loop(ctx context.Context) {
	defer close(b.done)
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.flush()
		case <-ctx.Done():
			b.flush()
			return
		}
	}
}

func (b *ActiveBatcher) flush() {
	b.mu.Lock()
	if len(b.ids) == 0 {
		b.mu.Unlock()
		return
	}
	ids := make([]string, 0, len(b.ids))
	for id := range b.ids {
		ids = append(ids, id)
	}
	b.ids = make(map[string]struct{})
	b.mu.Unlock()

	if err := b.store.BatchTouchUsersLastActive(ids); err != nil {
		slog.Error("batch touch users last active failed", "count", len(ids), "error", err)
	}
}
