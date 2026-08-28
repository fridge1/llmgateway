INSERT INTO model_pricing (model_name, input_price, output_price, cached_input_price,
    billing_type, is_active, pricing_tiers, updated_at)
VALUES (
    'pa/glm-5.1',
    6.000000, 24.000000, 1.300000,
    'token', true,
    '[{"min_tokens":1,"max_tokens":32768,"input_price":6,"output_price":24,"cached_input_price":1.3},{"min_tokens":32768,"max_tokens":204800,"input_price":8,"output_price":28,"cached_input_price":2}]'::jsonb,
    NOW()
)
ON CONFLICT (model_name) DO NOTHING;
