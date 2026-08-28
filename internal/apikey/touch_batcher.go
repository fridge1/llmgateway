package apikey

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// TouchStore is the subset of store.Store needed by TouchBatcher.
type TouchStore interface {
	BatchTouchAPIKeysLastUsed(ids []string) error
}

// TouchBatcher batches TouchAPIKeyLastUsed calls into periodic bulk updates.
type TouchBatcher struct {
	store    TouchStore
	interval time.Duration

	mu  sync.Mutex
	ids map[string]struct{}

	cancel context.CancelFunc
	done   chan struct{}
}

// NewTouchBatcher creates a batcher that flushes every interval.
func NewTouchBatcher(store TouchStore, interval time.Duration) *TouchBatcher {
	ctx, cancel := context.WithCancel(context.Background())
	b := &TouchBatcher{
		store:    store,
		interval: interval,
		ids:      make(map[string]struct{}),
		cancel:   cancel,
		done:     make(chan struct{}),
	}
	go b.loop(ctx)
	return b
}

// Touch enqueues an API key ID for a batched last_used_at update.
func (b *TouchBatcher) Touch(id string) {
	b.mu.Lock()
	b.ids[id] = struct{}{}
	b.mu.Unlock()
}

// Stop flushes remaining IDs and stops the background goroutine.
func (b *TouchBatcher) Stop() {
	b.cancel()
	<-b.done
}

func (b *TouchBatcher) loop(ctx context.Context) {
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

func (b *TouchBatcher) flush() {
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

	if err := b.store.BatchTouchAPIKeysLastUsed(ids); err != nil {
		slog.Error("batch touch api keys failed", "count", len(ids), "error", err)
	}
}
