// Package public hosts unauthenticated handlers used by the marketing
// landing page (HomePage). All endpoints are read-only aggregates, cached
// in-process to keep raw DB load bounded against arbitrary public traffic.
package public

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhulang/llm-gateway/internal/store"
)

const (
	plansTTL   = 60 * time.Second
	statsTTL   = 5 * time.Minute
	modelsTTL  = 60 * time.Second
	pricingTTL = 60 * time.Second
)

// Handler exposes public landing-page endpoints with in-memory caching.
type Handler struct {
	store store.Store

	plans   atomic.Pointer[plansSnapshot]
	stats   atomic.Pointer[statsSnapshot]
	models  atomic.Pointer[modelsSnapshot]
	pricing atomic.Pointer[pricingSnapshot]

	plansMu   sync.Mutex
	statsMu   sync.Mutex
	modelsMu  sync.Mutex
	pricingMu sync.Mutex
}

func New(s store.Store) *Handler {
	return &Handler{store: s}
}

// ---------- DTOs ----------

type planDTO struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	DisplayName     string  `json:"display_name"`
	Description     string  `json:"description"`
	MonthlyPriceCNY float64 `json:"monthly_price_cny"`
	QuotaAmountCNY  float64 `json:"quota_amount_cny"`
	DurationDays    int     `json:"duration_days"`
	SortOrder       int     `json:"sort_order"`
	Recommended     bool    `json:"recommended"`
}

type plansSnapshot struct {
	plans  []planDTO
	loaded time.Time
}

type statsDTO struct {
	TotalUsers       int64     `json:"total_users"`
	TotalEnterprises int64     `json:"total_enterprises"`
	TotalRequests    int64     `json:"total_requests"`
	TotalTokens      int64     `json:"total_tokens"`
	GeneratedAt      time.Time `json:"generated_at"`
}

type statsSnapshot struct {
	data   statsDTO
	loaded time.Time
}

type modelDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type providerGroupDTO struct {
	Provider string     `json:"provider"`
	Models   []modelDTO `json:"models"`
}

type modelsSnapshot struct {
	providers []providerGroupDTO
	loaded    time.Time
}

type pricingItemDTO struct {
	ModelName            string              `json:"model_name"`
	DisplayName          string              `json:"display_name"`
	InputPrice           float64             `json:"input_price"`
	OutputPrice          float64             `json:"output_price"`
	CachedInputPrice     float64             `json:"cached_input_price"`
	CacheCreationPrice   float64             `json:"cache_creation_price"`
	CacheCreation1hPrice float64             `json:"cache_creation_1h_price"`
	BillingType          string              `json:"billing_type"`
	PricingTiers         []store.PricingTier `json:"pricing_tiers,omitempty"`
}

type pricingProviderGroupDTO struct {
	Provider string           `json:"provider"`
	Items    []pricingItemDTO `json:"items"`
}

type pricingSnapshot struct {
	providers []pricingProviderGroupDTO
	loaded    time.Time
}

// ---------- Handlers ----------

// HandleListPlans returns active subscription plans with a 60s cache.
func (h *Handler) HandleListPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap := h.plans.Load()
	if snap == nil || time.Since(snap.loaded) > plansTTL {
		var err error
		snap, err = h.refreshPlans()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, plansTTL, map[string]any{"plans": snap.plans})
}

// HandleStats returns landing-page totals with a 5-minute cache.
// On TTL expiry, returns the stale value while a background goroutine
// refreshes (best-effort) so a single slow query never stalls the response.
func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap := h.stats.Load()
	if snap == nil {
		var err error
		snap, err = h.refreshStats()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else if time.Since(snap.loaded) > statsTTL {
		go func() { _, _ = h.refreshStats() }()
	}
	writeJSON(w, statsTTL, snap.data)
}

// HandleListModels returns models grouped by upstream provider with a 60s cache.
func (h *Handler) HandleListModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap := h.models.Load()
	if snap == nil || time.Since(snap.loaded) > modelsTTL {
		var err error
		snap, err = h.refreshModels()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, modelsTTL, map[string]any{"providers": snap.providers})
}

// HandleListPricing returns active model pricing grouped by provider, with a 60s cache.
func (h *Handler) HandleListPricing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap := h.pricing.Load()
	if snap == nil || time.Since(snap.loaded) > pricingTTL {
		var err error
		snap, err = h.refreshPricing()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, pricingTTL, map[string]any{"providers": snap.providers})
}

// ---------- Cache refresh ----------

func (h *Handler) refreshPlans() (*plansSnapshot, error) {
	h.plansMu.Lock()
	defer h.plansMu.Unlock()
	if cur := h.plans.Load(); cur != nil && time.Since(cur.loaded) <= plansTTL {
		return cur, nil
	}

	plans, err := h.store.ListSubscriptionPlans()
	if err != nil {
		return nil, err
	}
	dtos := make([]planDTO, 0, len(plans))
	for _, p := range plans {
		if p.Status != "active" {
			continue
		}
		dtos = append(dtos, planDTO{
			ID:              p.ID,
			Name:            p.Name,
			DisplayName:     p.DisplayName,
			Description:     p.Description,
			MonthlyPriceCNY: p.MonthlyPriceCNY,
			QuotaAmountCNY:  p.QuotaAmountCNY,
			DurationDays:    p.DurationDays,
			SortOrder:       p.SortOrder,
			Recommended:     p.Name == "pro",
		})
	}
	snap := &plansSnapshot{plans: dtos, loaded: time.Now()}
	h.plans.Store(snap)
	return snap, nil
}

func (h *Handler) refreshStats() (*statsSnapshot, error) {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	if cur := h.stats.Load(); cur != nil && time.Since(cur.loaded) <= statsTTL {
		return cur, nil
	}

	users, err := h.store.CountActiveUsers()
	if err != nil {
		return nil, err
	}
	tenants, err := h.store.CountEnterpriseTenants()
	if err != nil {
		return nil, err
	}
	totals, err := h.store.GetPublicUsageTotals()
	if err != nil {
		return nil, err
	}
	snap := &statsSnapshot{
		data: statsDTO{
			TotalUsers:       users,
			TotalEnterprises: tenants,
			TotalRequests:    totals.TotalRequests,
			TotalTokens:      totals.TotalTokens,
			GeneratedAt:      time.Now(),
		},
		loaded: time.Now(),
	}
	h.stats.Store(snap)
	return snap, nil
}

func (h *Handler) refreshModels() (*modelsSnapshot, error) {
	h.modelsMu.Lock()
	defer h.modelsMu.Unlock()
	if cur := h.models.Load(); cur != nil && time.Since(cur.loaded) <= modelsTTL {
		return cur, nil
	}

	models, err := h.store.ListModels()
	if err != nil {
		return nil, err
	}
	// Group by the first upstream's provider; fall back to "other".
	byProvider := make(map[string][]modelDTO)
	for _, m := range models {
		provider := "other"
		if len(m.Upstreams) > 0 && m.Upstreams[0].Provider != "" {
			provider = m.Upstreams[0].Provider
		}
		byProvider[provider] = append(byProvider[provider], modelDTO{
			ID:          m.Name,
			DisplayName: firstNonEmpty(m.DisplayName, m.Name),
		})
	}

	groups := make([]providerGroupDTO, 0, len(byProvider))
	for p, ms := range byProvider {
		sort.Slice(ms, func(i, j int) bool { return ms[i].DisplayName < ms[j].DisplayName })
		groups = append(groups, providerGroupDTO{Provider: p, Models: ms})
	}
	sort.Slice(groups, func(i, j int) bool {
		return providerOrder(groups[i].Provider) < providerOrder(groups[j].Provider)
	})

	snap := &modelsSnapshot{providers: groups, loaded: time.Now()}
	h.models.Store(snap)
	return snap, nil
}

func (h *Handler) refreshPricing() (*pricingSnapshot, error) {
	h.pricingMu.Lock()
	defer h.pricingMu.Unlock()
	if cur := h.pricing.Load(); cur != nil && time.Since(cur.loaded) <= pricingTTL {
		return cur, nil
	}

	prices, err := h.store.ListActivePricing()
	if err != nil {
		return nil, err
	}

	models, err := h.store.ListModels()
	if err != nil {
		return nil, err
	}
	type modelMeta struct {
		provider    string
		displayName string
	}
	metaByName := make(map[string]modelMeta, len(models))
	for _, m := range models {
		provider := "other"
		if len(m.Upstreams) > 0 && m.Upstreams[0].Provider != "" {
			provider = m.Upstreams[0].Provider
		}
		metaByName[m.Name] = modelMeta{
			provider:    provider,
			displayName: firstNonEmpty(m.DisplayName, m.Name),
		}
	}

	byProvider := make(map[string][]pricingItemDTO)
	for _, p := range prices {
		meta, ok := metaByName[p.ModelName]
		if !ok {
			meta = modelMeta{provider: "other", displayName: p.ModelName}
		}
		byProvider[meta.provider] = append(byProvider[meta.provider], pricingItemDTO{
			ModelName:            p.ModelName,
			DisplayName:          meta.displayName,
			InputPrice:           p.InputPrice,
			OutputPrice:          p.OutputPrice,
			CachedInputPrice:     p.CachedInputPrice,
			CacheCreationPrice:   p.CacheCreationPrice,
			CacheCreation1hPrice: p.CacheCreation1hPrice,
			BillingType:          p.BillingType,
			PricingTiers:         p.PricingTiers,
		})
	}

	groups := make([]pricingProviderGroupDTO, 0, len(byProvider))
	for prov, items := range byProvider {
		sort.Slice(items, func(i, j int) bool { return items[i].DisplayName < items[j].DisplayName })
		groups = append(groups, pricingProviderGroupDTO{Provider: prov, Items: items})
	}
	sort.Slice(groups, func(i, j int) bool {
		return providerOrder(groups[i].Provider) < providerOrder(groups[j].Provider)
	})

	snap := &pricingSnapshot{providers: groups, loaded: time.Now()}
	h.pricing.Store(snap)
	return snap, nil
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, maxAge time.Duration, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age="+itoa(int(maxAge.Seconds())))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// providerOrder defines the display priority on the landing page.
// Known providers come first in a fixed order; unknowns sort to the end.
func providerOrder(p string) int {
	switch p {
	case "openai":
		return 0
	case "anthropic":
		return 1
	case "google", "gemini":
		return 2
	default:
		return 100
	}
}
