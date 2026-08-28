package apikey

import (
	"container/list"
	"sync"
	"time"

	"github.com/zhulang/llm-gateway/internal/store"
)

// CachedAuth holds cached authentication info including user and API key ID.
type CachedAuth struct {
	User     *store.User
	APIKeyID string
	PlanID   *int // 非空时该 key 仅可访问该套餐内模型
	// MemberTenantID is the tenant this user belongs to (earliest-joined active
	// membership), or "" if none. When set, a personal-key request is billed to
	// and routed through this tenant.
	MemberTenantID string
}

// cacheEntry holds a cached auth with expiration time and LRU tracking.
type cacheEntry struct {
	key       string
	auth      *CachedAuth
	expiresAt time.Time
	element   *list.Element
}

// Cache is a concurrent-safe LRU cache from API key hash to *CachedAuth
// with a maximum capacity and TTL-based expiration.
type Cache struct {
	mu       sync.RWMutex
	entries  map[string]*cacheEntry
	lru      *list.List
	capacity int
	ttl      time.Duration
	stopCh   chan struct{}
}

// NewCache creates a new Cache with the given capacity and TTL.
func NewCache(capacity int, ttl time.Duration) *Cache {
	c := &Cache{
		entries:  make(map[string]*cacheEntry, capacity),
		lru:      list.New(),
		capacity: capacity,
		ttl:      ttl,
		stopCh:   make(chan struct{}),
	}
	go c.cleanup()
	return c
}

// Stop terminates the background cleanup goroutine.
func (c *Cache) Stop() {
	close(c.stopCh)
}

// Get retrieves cached auth from the cache by key hash. Returns nil if not found or expired.
func (c *Cache) Get(keyHash string) *CachedAuth {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[keyHash]
	if !ok {
		return nil
	}
	if time.Now().After(entry.expiresAt) {
		c.removeLocked(keyHash, entry)
		return nil
	}
	// Move to front (most recently used)
	c.lru.MoveToFront(entry.element)
	return entry.auth
}

// Set stores cached auth in the cache by key hash. Evicts LRU entry if at capacity.
func (c *Cache) Set(keyHash string, auth *CachedAuth) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing entry
	if existing, ok := c.entries[keyHash]; ok {
		existing.auth = auth
		existing.expiresAt = time.Now().Add(c.ttl)
		c.lru.MoveToFront(existing.element)
		return
	}

	// Evict LRU entry if at capacity
	if len(c.entries) >= c.capacity {
		c.evictLRULocked()
	}

	elem := c.lru.PushFront(keyHash)
	c.entries[keyHash] = &cacheEntry{
		key:       keyHash,
		auth:      auth,
		expiresAt: time.Now().Add(c.ttl),
		element:   elem,
	}
}

// Invalidate removes an entry from the cache.
func (c *Cache) Invalidate(keyHash string) {
	c.mu.Lock()
	if entry, ok := c.entries[keyHash]; ok {
		c.removeLocked(keyHash, entry)
	}
	c.mu.Unlock()
}

// InvalidateUser removes all cache entries belonging to the given user.
func (c *Cache) InvalidateUser(userID string) {
	c.mu.Lock()
	for k, e := range c.entries {
		if e.auth != nil && e.auth.User != nil && e.auth.User.ID == userID {
			c.removeLocked(k, e)
		}
	}
	c.mu.Unlock()
}

func (c *Cache) removeLocked(key string, entry *cacheEntry) {
	c.lru.Remove(entry.element)
	delete(c.entries, key)
}

func (c *Cache) evictLRULocked() {
	back := c.lru.Back()
	if back == nil {
		return
	}
	key := back.Value.(string)
	if entry, ok := c.entries[key]; ok {
		c.removeLocked(key, entry)
	}
}

// cleanup periodically removes expired entries.
func (c *Cache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, e := range c.entries {
				if now.After(e.expiresAt) {
					c.removeLocked(k, e)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}

