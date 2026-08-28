package router

import (
	"strings"

	"github.com/zhulang/llm-gateway/internal/balancer"
	"github.com/zhulang/llm-gateway/internal/circuit"
	"github.com/zhulang/llm-gateway/internal/config"
)

// modelEntry holds the canonical name, display name, and upstreams for a logical model.
type modelEntry struct {
	canonicalName string
	displayName   string
	upstreams     []balancer.Upstream
}

// ModelInfo holds metadata about a resolved model.
type ModelInfo struct {
	CanonicalName string
	DisplayName   string
}

// Router maps model names to upstream pools.
type Router struct {
	table map[string]*modelEntry
	// tenantTable maps tenantID → canonical model name → tenant-exclusive
	// upstream pool. A hit here replaces the global pool entirely (no fallback).
	tenantTable map[string]map[string]*modelEntry
}

// NewFromConfig builds a Router from cfg, creating a circuit breaker per upstream.
func NewFromConfig(cfg *config.Config) *Router {
	return NewFromConfigWithBreakers(cfg, nil)
}

// NewFromConfigWithBreakers builds a Router from cfg, reusing existing circuit breakers
// where the upstream URL matches. This preserves failure state across router rebuilds.
func NewFromConfigWithBreakers(cfg *config.Config, existing *Router) *Router {
	table := make(map[string]*modelEntry, len(cfg.Models))
	cb := cfg.CircuitBreaker

	// Build a lookup of existing breakers by model+URL for reuse.
	// Tenant upstream breakers use a "t:<tenantID>|" prefix so a tenant's
	// breaker for the same base URL stays independent from the global one.
	existingBreakers := make(map[string]*circuit.Breaker)
	if existing != nil {
		for _, entry := range existing.table {
			for _, u := range entry.upstreams {
				existingBreakers[entry.canonicalName+"|"+u.Config.BaseURL] = u.Breaker
			}
		}
		for tenantID, tm := range existing.tenantTable {
			for _, entry := range tm {
				for _, u := range entry.upstreams {
					existingBreakers["t:"+tenantID+"|"+entry.canonicalName+"|"+u.Config.BaseURL] = u.Breaker
				}
			}
		}
	}

	newBreaker := func(key string) *circuit.Breaker {
		if b, ok := existingBreakers[key]; ok {
			return b
		}
		return circuit.NewBreaker(
			cb.FailureThreshold,
			cb.RecoveryTimeout,
			cb.HalfOpenMaxRequests,
		)
	}

	for _, m := range cfg.Models {
		upstreams := make([]balancer.Upstream, len(m.Upstreams))
		for i, u := range m.Upstreams {
			upstreams[i] = balancer.Upstream{
				Config:  u,
				Breaker: newBreaker(m.Name + "|" + u.BaseURL),
			}
		}
		entry := &modelEntry{
			canonicalName: m.Name,
			displayName:   m.DisplayName,
			upstreams:     upstreams,
		}
		table[m.Name] = entry
		if m.DisplayName != "" && m.DisplayName != m.Name {
			table[m.DisplayName] = entry
		}
	}

	// Tenant-specific overlays: keyed by canonical model name only; alias and
	// prefix resolution happens through the global table in GetUpstreamsForTenant.
	tenantTable := make(map[string]map[string]*modelEntry)
	for _, tm := range cfg.TenantModels {
		upstreams := make([]balancer.Upstream, len(tm.Upstreams))
		for i, u := range tm.Upstreams {
			upstreams[i] = balancer.Upstream{
				Config:  u,
				Breaker: newBreaker("t:" + tm.TenantID + "|" + tm.ModelName + "|" + u.BaseURL),
			}
		}
		displayName := tm.ModelName
		if global, ok := table[tm.ModelName]; ok {
			displayName = global.displayName
		}
		if tenantTable[tm.TenantID] == nil {
			tenantTable[tm.TenantID] = make(map[string]*modelEntry)
		}
		tenantTable[tm.TenantID][tm.ModelName] = &modelEntry{
			canonicalName: tm.ModelName,
			displayName:   displayName,
			upstreams:     upstreams,
		}
	}

	return &Router{table: table, tenantTable: tenantTable}
}

// NormalizeModelName strips the "gw/" prefix if present.
func NormalizeModelName(model string) string {
	if strings.HasPrefix(model, "gw/") {
		return model[3:]
	}
	return model
}

// GetUpstreams looks up a model by name and returns its upstreams, model info,
// and whether the model was found.
func (r *Router) GetUpstreams(model string) ([]balancer.Upstream, ModelInfo, bool) {
	entry, ok := r.table[model]
	if !ok {
		// Strip "gw/" prefix and retry.
		if strings.HasPrefix(model, "gw/") {
			entry, ok = r.table[model[3:]]
		}
		if !ok {
			stripped := NormalizeModelName(model)
			// Strip trailing "[...]" suffix for GLM models (e.g. "glm-5.1[1m]" → "glm-5.1").
			if strings.HasPrefix(stripped, "glm") {
				if idx := strings.IndexByte(stripped, '['); idx > 0 {
					entry, ok = r.table[stripped[:idx]]
				}
			}
		}
		if !ok {
			return nil, ModelInfo{}, false
		}
	}
	return entry.upstreams, ModelInfo{CanonicalName: NormalizeModelName(entry.canonicalName), DisplayName: entry.displayName}, true
}

// GetUpstreamsForTenant resolves a model like GetUpstreams, but if the tenant
// has an upstream override for the resolved model, the override pool is
// returned exclusively (no fallback to the global pool on exhaustion).
// An empty tenantID behaves exactly like GetUpstreams.
func (r *Router) GetUpstreamsForTenant(tenantID, model string) ([]balancer.Upstream, ModelInfo, bool) {
	ups, info, ok := r.GetUpstreams(model)
	if tenantID == "" {
		return ups, info, ok
	}
	tm := r.tenantTable[tenantID]
	if tm == nil {
		return ups, info, ok
	}
	// Global hit already resolved gw/ prefix, display-name alias and glm
	// suffix into the canonical name; for orphan overrides (model removed
	// from the global table) fall back to the normalized request name.
	name := info.CanonicalName
	if !ok {
		name = NormalizeModelName(model)
	}
	if entry, hit := tm[name]; hit {
		return entry.upstreams, ModelInfo{CanonicalName: NormalizeModelName(entry.canonicalName), DisplayName: entry.displayName}, true
	}
	return ups, info, ok
}

// ListModels returns all canonical model names registered in the router.
// Aliases (display_name entries) are excluded to avoid duplicates.
func (r *Router) ListModels() []string {
	seen := make(map[string]bool, len(r.table))
	names := make([]string, 0, len(r.table))
	for _, entry := range r.table {
		if seen[entry.canonicalName] {
			continue
		}
		seen[entry.canonicalName] = true
		names = append(names, entry.canonicalName)
	}
	return names
}

// Entries returns the routing table as a map of canonical model name to upstreams,
// suitable for status reporting. Aliases are excluded to avoid duplicates.
func (r *Router) Entries() map[string][]balancer.Upstream {
	seen := make(map[string]bool, len(r.table))
	out := make(map[string][]balancer.Upstream, len(r.table))
	for _, entry := range r.table {
		if seen[entry.canonicalName] {
			continue
		}
		seen[entry.canonicalName] = true
		out[entry.canonicalName] = entry.upstreams
	}
	return out
}

// UpstreamHealth returns the total number of unique upstreams and how many are healthy
// (circuit breaker allows requests).
func (r *Router) UpstreamHealth() (total, healthy int) {
	seen := make(map[string]bool)
	for _, entry := range r.table {
		for _, u := range entry.upstreams {
			key := entry.canonicalName + "|" + u.Config.BaseURL
			if seen[key] {
				continue
			}
			seen[key] = true
			total++
			if u.Breaker.AllowRequest() {
				healthy++
			}
		}
	}
	return
}
