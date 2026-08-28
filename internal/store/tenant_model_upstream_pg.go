package store

import (
	"fmt"

	"github.com/lib/pq"
)

const tenantModelUpstreamSelect = `id, tenant_id, model_name, provider, protocol, protocols,
	upstream_provider, upstream_name, base_url, api_key, model_override,
	weight, sort_order, created_at, updated_at`

func scanTenantModelUpstream(rows interface {
	Scan(dest ...any) error
}) (TenantModelUpstream, error) {
	var u TenantModelUpstream
	err := rows.Scan(
		&u.ID, &u.TenantID, &u.ModelName, &u.Provider, &u.Protocol, pq.Array(&u.Protocols),
		&u.UpstreamProvider, &u.UpstreamName, &u.BaseURL, &u.APIKey, &u.ModelOverride,
		&u.Weight, &u.SortOrder, &u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

// ListTenantModelUpstreams returns all upstream overrides for a tenant,
// grouped by model via ordering.
func (s *PgStore) ListTenantModelUpstreams(tenantID string) ([]TenantModelUpstream, error) {
	rows, err := s.db.Query(
		`SELECT `+tenantModelUpstreamSelect+` FROM tenant_model_upstreams
		 WHERE tenant_id = $1 ORDER BY model_name, sort_order`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list tenant model upstreams: %w", err)
	}
	defer rows.Close()

	ups := []TenantModelUpstream{}
	for rows.Next() {
		u, err := scanTenantModelUpstream(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan tenant model upstream: %w", err)
		}
		ups = append(ups, u)
	}
	return ups, rows.Err()
}

// ListAllTenantModelUpstreams returns every tenant upstream override across
// all tenants, used to build the router's tenant overlay.
func (s *PgStore) ListAllTenantModelUpstreams() ([]TenantModelUpstream, error) {
	rows, err := s.db.Query(
		`SELECT ` + tenantModelUpstreamSelect + ` FROM tenant_model_upstreams
		 ORDER BY tenant_id, model_name, sort_order`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list all tenant model upstreams: %w", err)
	}
	defer rows.Close()

	ups := []TenantModelUpstream{}
	for rows.Next() {
		u, err := scanTenantModelUpstream(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan tenant model upstream: %w", err)
		}
		ups = append(ups, u)
	}
	return ups, rows.Err()
}

// ReplaceTenantModelUpstreams atomically replaces the upstream override list
// for one tenant+model (delete + ordered reinsert, mirroring UpdateModel).
func (s *PgStore) ReplaceTenantModelUpstreams(tenantID, modelName string, upstreams []TenantModelUpstream) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`DELETE FROM tenant_model_upstreams WHERE tenant_id = $1 AND model_name = $2`,
		tenantID, modelName,
	)
	if err != nil {
		return fmt.Errorf("store: delete tenant model upstreams: %w", err)
	}

	for i, u := range upstreams {
		_, err := tx.Exec(
			`INSERT INTO tenant_model_upstreams (tenant_id, model_name, provider, protocol, protocols,
			   upstream_provider, upstream_name, base_url, api_key, model_override, weight, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			tenantID, modelName, u.Provider, u.Protocol, pq.Array(u.Protocols),
			u.UpstreamProvider, u.UpstreamName, u.BaseURL, u.APIKey, u.ModelOverride, u.Weight, i,
		)
		if err != nil {
			return fmt.Errorf("store: insert tenant model upstream: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit: %w", err)
	}
	return nil
}

// DeleteTenantModelUpstreams removes all upstream overrides for one tenant+model.
func (s *PgStore) DeleteTenantModelUpstreams(tenantID, modelName string) error {
	_, err := s.db.Exec(
		`DELETE FROM tenant_model_upstreams WHERE tenant_id = $1 AND model_name = $2`,
		tenantID, modelName,
	)
	if err != nil {
		return fmt.Errorf("store: delete tenant model upstreams: %w", err)
	}
	return nil
}
