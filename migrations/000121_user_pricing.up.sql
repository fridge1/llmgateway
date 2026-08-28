CREATE TABLE user_pricing (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_name VARCHAR(255) NOT NULL,
    input_price NUMERIC(20,8) NOT NULL DEFAULT 0,
    output_price NUMERIC(20,8) NOT NULL DEFAULT 0,
    cached_input_price NUMERIC(20,8) NOT NULL DEFAULT 0,
    cache_creation_price NUMERIC(20,8) NOT NULL DEFAULT 0,
    cache_creation_1h_price NUMERIC(20,8) NOT NULL DEFAULT 0,
    billing_type VARCHAR(20) NOT NULL DEFAULT 'token',
    is_active BOOLEAN NOT NULL DEFAULT true,
    pricing_tiers JSONB,
    discount_rate NUMERIC(6,4),
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, model_name)
);

CREATE INDEX idx_user_pricing_user ON user_pricing(user_id);
CREATE INDEX idx_user_pricing_model ON user_pricing(model_name);

COMMENT ON COLUMN user_pricing.discount_rate IS '折扣率(0-1]，非空时实际价=全局价×该值；为空时回退使用本表绝对单价(历史兼容)';
