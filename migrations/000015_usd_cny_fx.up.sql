-- USD 基准价 + 实时汇率换算为人民币（计费字段 input_price 等为 CNY / 百万 tokens）
CREATE TABLE IF NOT EXISTS fx_rates (
    id SMALLINT PRIMARY KEY DEFAULT 1,
    usd_cny DECIMAL(14, 8) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fx_rates_single_row CHECK (id = 1)
);

INSERT INTO fx_rates (id, usd_cny) VALUES (1, 7.20000000)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS input_price_usd DECIMAL(10, 6);
ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS output_price_usd DECIMAL(10, 6);
ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS cached_input_price_usd DECIMAL(10, 6) NOT NULL DEFAULT 0;

-- 将现有数值视为美元基准价（与历史种子注释一致），并按下表初始汇率生成人民币计费价
UPDATE model_pricing SET
    input_price_usd = input_price,
    output_price_usd = output_price,
    cached_input_price_usd = COALESCE(cached_input_price, 0)
WHERE input_price_usd IS NULL;

ALTER TABLE model_pricing ALTER COLUMN input_price_usd SET NOT NULL;
ALTER TABLE model_pricing ALTER COLUMN output_price_usd SET NOT NULL;

UPDATE model_pricing SET
    input_price = input_price_usd * (SELECT usd_cny FROM fx_rates WHERE id = 1),
    output_price = output_price_usd * (SELECT usd_cny FROM fx_rates WHERE id = 1),
    cached_input_price = cached_input_price_usd * (SELECT usd_cny FROM fx_rates WHERE id = 1),
    updated_at = NOW();
