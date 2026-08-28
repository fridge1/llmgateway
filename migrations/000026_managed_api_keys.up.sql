CREATE TABLE managed_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    key_prefix VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    quota DECIMAL(12,4) NOT NULL DEFAULT 0,
    created_by UUID,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_managed_api_keys_hash ON managed_api_keys(key_hash);
CREATE INDEX idx_managed_api_keys_status ON managed_api_keys(status);

CREATE TABLE managed_api_key_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id UUID NOT NULL REFERENCES managed_api_keys(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    amount DECIMAL(12,4) NOT NULL,
    quota_after DECIMAL(12,4) NOT NULL,
    model VARCHAR(100),
    description TEXT,
    request_id VARCHAR(100),
    related_key_id UUID,
    prompt_tokens INT,
    completion_tokens INT,
    cache_read_tokens INT,
    cache_creation_tokens INT,
    cache_creation_5m_tokens INT,
    cache_creation_1h_tokens INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_managed_key_tx_key_id ON managed_api_key_transactions(key_id, created_at DESC);
