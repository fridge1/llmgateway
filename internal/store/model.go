package store

import (
	"time"

	"github.com/zhulang/llm-gateway/internal/config"
)

// Model represents a model with its upstreams.
type Model struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name"`
	Category    string     `json:"category"`
	Upstreams   []Upstream `json:"upstreams"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Upstream represents a single upstream provider.
type Upstream struct {
	ID               int64    `json:"id"`
	ModelID          int64    `json:"model_id"`
	Provider         string   `json:"provider"`
	Protocol         string   `json:"protocol"`
	Protocols        []string `json:"protocols"`
	UpstreamProvider string   `json:"upstream_provider"`
	UpstreamName     string   `json:"upstream_name"`
	BaseURL          string   `json:"base_url"`
	APIKey           string   `json:"api_key"`
	ModelOverride    string   `json:"model_override"`
	Weight           int      `json:"weight"`
	SortOrder        int      `json:"sort_order"`
}

// ToConfig converts all store models to a config.Config models slice,
// merging with the base config's server settings.
func ToConfig(s Store, baseCfg *config.Config) (*config.Config, error) {
	models, err := s.ListModels()
	if err != nil {
		return nil, err
	}

	cfgModels := make([]config.ModelConfig, len(models))
	for i, m := range models {
		upstreams := make([]config.UpstreamConfig, len(m.Upstreams))
		for j, u := range m.Upstreams {
			upstreams[j] = config.UpstreamConfig{
				Provider:         u.Provider,
				Protocol:         u.Protocol,
				Protocols:        u.Protocols,
				UpstreamProvider: u.UpstreamProvider,
				UpstreamName:     u.UpstreamName,
				BaseURL:          u.BaseURL,
				APIKey:           u.APIKey,
				ModelOverride:    u.ModelOverride,
				Weight:           u.Weight,
			}
		}
		cfgModels[i] = config.ModelConfig{
			Name:        m.Name,
			DisplayName: m.DisplayName,
			Upstreams:   upstreams,
		}
	}

	// Tenant-specific upstream overrides (DB-only; routed via the tenant
	// overlay in the router).
	tenantUps, err := s.ListAllTenantModelUpstreams()
	if err != nil {
		return nil, err
	}
	var tenantModels []config.TenantModelConfig
	for _, u := range tenantUps {
		n := len(tenantModels)
		if n == 0 || tenantModels[n-1].TenantID != u.TenantID || tenantModels[n-1].ModelName != u.ModelName {
			tenantModels = append(tenantModels, config.TenantModelConfig{
				TenantID:  u.TenantID,
				ModelName: u.ModelName,
			})
			n++
		}
		tenantModels[n-1].Upstreams = append(tenantModels[n-1].Upstreams, config.UpstreamConfig{
			Provider:         u.Provider,
			Protocol:          u.Protocol,
			Protocols:         u.Protocols,
			UpstreamProvider:  u.UpstreamProvider,
			UpstreamName:      u.UpstreamName,
			BaseURL:           u.BaseURL,
			APIKey:            u.APIKey,
			ModelOverride:     u.ModelOverride,
			Weight:            u.Weight,
		})
	}

	return &config.Config{
		Server:         baseCfg.Server,
		Models:         cfgModels,
		TenantModels:   tenantModels,
		CircuitBreaker: baseCfg.CircuitBreaker,
		Database:       baseCfg.Database,
		Admin:          baseCfg.Admin,
		SMS:            baseCfg.SMS,
		Payment:        baseCfg.Payment,
		Promotion:      baseCfg.Promotion,
		Retention:      baseCfg.Retention,
		TOS:            baseCfg.TOS,
		Billing:        baseCfg.Billing,
	}, nil
}
