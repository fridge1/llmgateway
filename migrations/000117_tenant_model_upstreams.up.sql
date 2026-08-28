CREATE TABLE tenant_model_upstreams (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    model_name VARCHAR(255) NOT NULL,
    provider VARCHAR(100) NOT NULL,
    protocol VARCHAR(20) NOT NULL DEFAULT '',
    upstream_provider VARCHAR(100) NOT NULL DEFAULT '',
    upstream_name VARCHAR(255) NOT NULL DEFAULT '',
    base_url TEXT NOT NULL,
    api_key TEXT NOT NULL,
    model_override VARCHAR(255) NOT NULL DEFAULT '',
    weight INT NOT NULL DEFAULT 1,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tenant_model_upstreams_tenant_model ON tenant_model_upstreams(tenant_id, model_name);
