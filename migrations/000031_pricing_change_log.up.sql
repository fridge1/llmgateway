CREATE TABLE IF NOT EXISTS pricing_change_logs (
    id BIGSERIAL PRIMARY KEY,
    model_name TEXT NOT NULL,
    change_type TEXT NOT NULL,  -- 'pricing_update', 'fx_rate_change'
    admin_user_id TEXT,
    old_values JSONB,
    new_values JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pricing_change_logs_model ON pricing_change_logs (model_name);
CREATE INDEX idx_pricing_change_logs_created ON pricing_change_logs (created_at DESC);
