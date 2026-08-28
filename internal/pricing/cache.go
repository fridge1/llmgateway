package pricing

import (
	"strings"
	"sync"
	"time"

	"github.com/zhulang/llm-gateway/internal/store"
)

// Cache is an in-memory cache for model pricing with TTL-based expiration.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*pricingEntry
	ttl     time.Duration
	store   store.Store
	stopCh  chan struct{}
}

type pricingEntry struct {
	pricing   *store.ModelPricing
	expiresAt time.Time
}

// NewCache creates a pricing cache with the given TTL and backing store.
func NewCache(s store.Store, ttl time.Duration) *Cache {
	c := &Cache{
		entries: make(map[string]*pricingEntry),
		ttl:     ttl,
		store:   s,
		stopCh:  make(chan struct{}),
	}
	go c.cleanup()
	return c
}

// GetPricing returns pricing for a model, using the cache when available.
func (c *Cache) GetPricing(modelName string) (*store.ModelPricing, error) {
	c.mu.RLock()
	e, ok := c.entries[modelName]
	c.mu.RUnlock()

	if ok && time.Now().Before(e.expiresAt) {
		return e.pricing, nil
	}

	// Cache miss or expired — fetch from DB.
	pricing, err := c.store.GetPricing(modelName)
	if err != nil {
		// Remove stale entry on error.
		c.mu.Lock()
		delete(c.entries, modelName)
		c.mu.Unlock()
		return nil, err
	}

	c.mu.Lock()
	c.entries[modelName] = &pricingEntry{
		pricing:   pricing,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return pricing, nil
}

// Invalidate removes a specific model from the cache.
func (c *Cache) Invalidate(modelName string) {
	c.mu.Lock()
	delete(c.entries, modelName)
	c.mu.Unlock()
}

// InvalidateAll clears the entire cache.
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	c.entries = make(map[string]*pricingEntry)
	c.mu.Unlock()
}

// Stop terminates the background cleanup goroutine.
func (c *Cache) Stop() {
	close(c.stopCh)
}

// GetTenantPricing returns tenant-specific pricing if it exists, falling back
// to global pricing. The second return value indicates whether tenant-specific
// pricing was used.
//
// When the tenant row carries a non-nil discount_rate, the effective pricing is
// computed as global pricing × rate (so it tracks global price changes). When
// discount_rate is nil, the tenant's absolute prices are used (legacy behavior).
func (c *Cache) GetTenantPricing(tenantID, modelName string) (*store.ModelPricing, bool, error) {
	tenantKey := tenantID + ":" + modelName

	c.mu.RLock()
	e, ok := c.entries[tenantKey]
	c.mu.RUnlock()

	if ok && time.Now().Before(e.expiresAt) {
		return e.pricing, true, nil
	}

	tp, err := c.store.GetTenantPricing(tenantID, modelName)
	if err == nil && tp != nil {
		var mp *store.ModelPricing
		if tp.DiscountRate != nil {
			// Discount mode: derive from current global pricing.
			gp, gerr := c.GetPricing(modelName)
			if gerr != nil {
				// Transient DB error fetching global pricing — do NOT cache a
				// zero-price fallback. Propagate the error so billing can retry.
				return nil, false, gerr
			}
			if gp == nil {
				// No global pricing to discount — fall back to absolute row.
				mp = tp.ToModelPricing()
			} else {
				mp = applyDiscount(gp, *tp.DiscountRate)
				mp.IsActive = tp.IsActive
			}		} else {
			mp = tp.ToModelPricing()
		}
		c.mu.Lock()
		c.entries[tenantKey] = &pricingEntry{
			pricing:   mp,
			expiresAt: time.Now().Add(c.ttl),
		}
		c.mu.Unlock()
		return mp, true, nil
	}

	gp, err := c.GetPricing(modelName)
	if err != nil {
		return nil, false, err
	}
	return gp, false, nil
}

// GetTenantPricingDetail resolves tenant pricing with discount metadata for
// display. It returns:
//   - effective: the price the tenant actually pays (discounted or absolute).
//   - rate: the discount rate when the tenant row uses discount mode; nil for
//     absolute pricing or when no tenant override exists.
//   - original: the global pricing the discount derives from, for showing a
//     struck-through original price; nil unless rate is non-nil.
//   - used: whether a tenant-specific override applied.
//
// Unlike GetTenantPricing, this bypasses the entry cache so the discount rate
// and original price stay in sync; it is meant for list-page loads, not the
// billing hot path.
func (c *Cache) GetTenantPricingDetail(tenantID, modelName string) (effective *store.ModelPricing, rate *float64, original *store.ModelPricing, used bool, err error) {
	tp, terr := c.store.GetTenantPricing(tenantID, modelName)
	if terr == nil && tp != nil {
		if tp.DiscountRate != nil {
			gp, gerr := c.GetPricing(modelName)
			if gerr == nil && gp != nil {
				return ApplyDiscount(gp, *tp.DiscountRate, tp.IsActive), tp.DiscountRate, gp, true, nil
			}
			// No global pricing to discount — fall back to absolute row.
			return tp.ToModelPricing(), nil, nil, true, nil
		}
		return tp.ToModelPricing(), nil, nil, true, nil
	}

	gp, gerr := c.GetPricing(modelName)
	if gerr != nil {
		return nil, nil, nil, false, gerr
	}
	return gp, nil, nil, false, nil
}

// applyDiscount returns a copy of global pricing with every price field scaled
// by rate. PricingTiers are scaled per tier. The original is not modified.
func applyDiscount(global *store.ModelPricing, rate float64) *store.ModelPricing {
	mp := *global // shallow copy of value fields
	mp.InputPrice = global.InputPrice * rate
	mp.OutputPrice = global.OutputPrice * rate
	mp.CachedInputPrice = global.CachedInputPrice * rate
	mp.CacheCreationPrice = global.CacheCreationPrice * rate
	mp.CacheCreation1hPrice = global.CacheCreation1hPrice * rate
	if len(global.PricingTiers) > 0 {
		tiers := make([]store.PricingTier, len(global.PricingTiers))
		for i, t := range global.PricingTiers {
			tiers[i] = store.PricingTier{
				MinTokens:        t.MinTokens,
				MaxTokens:        t.MaxTokens,
				InputPrice:       t.InputPrice * rate,
				OutputPrice:      t.OutputPrice * rate,
				CachedInputPrice: t.CachedInputPrice * rate,
			}
		}
		mp.PricingTiers = tiers
	}
	return &mp
}

// ApplyDiscount returns global pricing scaled by rate, with IsActive overridden.
// Exported for callers that compute tenant discount pricing without the cache.
func ApplyDiscount(global *store.ModelPricing, rate float64, isActive bool) *store.ModelPricing {
	mp := applyDiscount(global, rate)
	mp.IsActive = isActive
	return mp
}

// GetUserPricing returns pricing for a user+model, using the cache.
// When the user row carries a non-nil discount_rate, the effective pricing is
// computed as global pricing × rate. When discount_rate is nil, the user's
// absolute prices are used (legacy behavior).
func (c *Cache) GetUserPricing(userID, modelName string) (*store.ModelPricing, bool, error) {
	userKey := "user:" + userID + ":" + modelName

	c.mu.RLock()
	e, ok := c.entries[userKey]
	c.mu.RUnlock()

	if ok && time.Now().Before(e.expiresAt) {
		return e.pricing, true, nil
	}

	up, err := c.store.GetUserPricing(userID, modelName)
	if err == nil && up != nil {
		var mp *store.ModelPricing
		if up.DiscountRate != nil {
			// Discount mode: derive from current global pricing.
			gp, gerr := c.GetPricing(modelName)
			if gerr != nil {
				// Transient DB error fetching global pricing — do NOT cache a
				// zero-price fallback. Propagate the error so billing can retry.
				return nil, false, gerr
			}
			if gp == nil {
				// No global pricing to discount — fall back to absolute row.
				mp = up.ToModelPricing()
			} else {
				mp = applyDiscount(gp, *up.DiscountRate)
				mp.IsActive = up.IsActive
			}
		} else {
			mp = up.ToModelPricing()
		}
		c.mu.Lock()
		c.entries[userKey] = &pricingEntry{
			pricing:   mp,
			expiresAt: time.Now().Add(c.ttl),
		}
		c.mu.Unlock()
		return mp, true, nil
	}

	return nil, false, err
}

// GetUserPricingDetail resolves user pricing with discount metadata for display.
// It returns:
//   - effective: the price the user actually pays (discounted or absolute).
//   - rate: the discount rate when the user row uses discount mode; nil for
//     absolute pricing or when no user override exists.
//   - original: the global pricing the discount derives from, for showing a
//     struck-through original price; nil unless rate is non-nil.
//   - used: whether a user-specific override applied.
//
// Unlike GetUserPricing, this bypasses the entry cache so the discount rate
// and original price stay in sync; it is meant for list-page loads, not the
// billing hot path.
func (c *Cache) GetUserPricingDetail(userID, modelName string) (effective *store.ModelPricing, rate *float64, original *store.ModelPricing, used bool, err error) {
	up, uerr := c.store.GetUserPricing(userID, modelName)
	if uerr == nil && up != nil {
		if up.DiscountRate != nil {
			// Discount mode: derive from current global pricing.
			gp, gerr := c.GetPricing(modelName)
			if gerr == nil && gp != nil {
				return ApplyDiscount(gp, *up.DiscountRate, up.IsActive), up.DiscountRate, gp, true, nil
			}
			// No global pricing to discount — fall back to absolute row.
			return up.ToModelPricing(), nil, nil, true, nil
		}
		return up.ToModelPricing(), nil, nil, true, nil
	}

	// No user pricing, return global pricing.
	gp, gerr := c.GetPricing(modelName)
	if gerr != nil {
		return nil, nil, nil, false, gerr
	}
	return gp, nil, nil, false, nil
}

// InvalidateModelAllTenants removes all cached tenant-derived entries for a model.
// Call this when a model's global pricing changes so discount-based tenant prices
// are recomputed from the new global price.
func (c *Cache) InvalidateModelAllTenants(modelName string) {
	suffix := ":" + modelName
	c.mu.Lock()
	for k := range c.entries {
		if strings.HasSuffix(k, suffix) && k != modelName {
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()
}

// InvalidateTenant removes a cached entry for a specific tenant+model.
func (c *Cache) InvalidateTenant(tenantID, modelName string) {
	c.mu.Lock()
	delete(c.entries, tenantID+":"+modelName)
	c.mu.Unlock()
}

// InvalidateAllTenant removes all cached entries for a tenant.
func (c *Cache) InvalidateAllTenant(tenantID string) {
	prefix := tenantID + ":"
	c.mu.Lock()
	for k := range c.entries {
		if strings.HasPrefix(k, prefix) {
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()
}

// InvalidateUser removes a cached entry for a specific user+model.
func (c *Cache) InvalidateUser(userID, modelName string) {
	c.mu.Lock()
	delete(c.entries, "user:"+userID+":"+modelName)
	c.mu.Unlock()
}

// InvalidateAllUser removes all cached entries for a user.
func (c *Cache) InvalidateAllUser(userID string) {
	prefix := "user:" + userID + ":"
	c.mu.Lock()
	for k := range c.entries {
		if strings.HasPrefix(k, prefix) {
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()
}

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
					delete(c.entries, k)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}
