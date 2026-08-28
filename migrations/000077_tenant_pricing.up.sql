CREATE TABLE tenant_pricing (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    model_name VARCHAR(255) NOT NULL,
    input_price NUMERIC(20,8) NOT NULL DEFAULT 0,
    output_price NUMERIC(20,8) NOT NULL DEFAULT 0,
    cached_input_price NUMERIC(20,8) NOT NULL DEFAULT 0,
    cache_creation_price NUMERIC(20,8) NOT NULL DEFAULT 0,
    cache_creation_1h_price NUMERIC(20,8) NOT NULL DEFAULT 0,
    billing_type VARCHAR(20) NOT NULL DEFAULT 'token',
    is_active BOOLEAN NOT NULL DEFAULT true,
    pricing_tiers JSONB,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, model_name)
);

CREATE INDEX idx_tenant_pricing_tenant ON tenant_pricing(tenant_id);
CREATE INDEX idx_tenant_pricing_model ON tenant_pricing(model_name);
