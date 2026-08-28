-- 恢复 fx_rates 表和 model_pricing 的 USD 列
CREATE TABLE IF NOT EXISTS fx_rates (
    id SMALLINT PRIMARY KEY DEFAULT 1,
    usd_cny DECIMAL(14, 8) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fx_rates_single_row CHECK (id = 1)
);

INSERT INTO fx_rates (id, usd_cny) VALUES (1, 7.20000000)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS input_price_usd DECIMAL(10, 6) NOT NULL DEFAULT 0;
ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS output_price_usd DECIMAL(10, 6) NOT NULL DEFAULT 0;
ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS cached_input_price_usd DECIMAL(10, 6) NOT NULL DEFAULT 0;
ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS cache_creation_price_usd DECIMAL(10, 6) NOT NULL DEFAULT 0;
ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS cache_creation_1h_price_usd DECIMAL(10, 6) NOT NULL DEFAULT 0;
