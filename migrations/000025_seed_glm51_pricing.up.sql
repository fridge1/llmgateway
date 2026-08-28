-- Seed GLM-5.1 tiered pricing.
-- Tier 1: 1 <= tokens < 32768  → input ¥6/M, output ¥24/M, cached ¥1.3/M
-- Tier 2: 32768 <= tokens < 204800 → input ¥8/M, output ¥28/M, cached ¥2/M
-- Base prices use tier-1 values; billing_type = "token".
INSERT INTO model_pricing (model_name, input_price, output_price, cached_input_price,
    input_price_usd, output_price_usd, cached_input_price_usd,
    billing_type, is_active, pricing_tiers, updated_at)
VALUES (
    'pa/glm-5.1',
    6.000000, 24.000000, 1.300000,
    0, 0, 0,
    'token', true,
    '[{"min_tokens":1,"max_tokens":32768,"input_price":6,"output_price":24,"cached_input_price":1.3},{"min_tokens":32768,"max_tokens":204800,"input_price":8,"output_price":28,"cached_input_price":2}]'::jsonb,
    NOW()
)
ON CONFLICT (model_name) DO UPDATE SET
    input_price = EXCLUDED.input_price,
    output_price = EXCLUDED.output_price,
    cached_input_price = EXCLUDED.cached_input_price,
    pricing_tiers = EXCLUDED.pricing_tiers,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();
