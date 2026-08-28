-- 添加图片生成模型的定价
-- input_price: 1K/2K 图片单价（CNY）
-- output_price: 4K 图片单价（CNY）

-- DALL-E 3 定价（示例价格，需根据实际成本调整）
INSERT INTO model_pricing (model_name, input_price, output_price, billing_type, is_active)
VALUES ('dall-e-3', 0.10, 0.20, 'image', true)
ON CONFLICT (model_name) DO UPDATE SET
    input_price = EXCLUDED.input_price,
    output_price = EXCLUDED.output_price,
    billing_type = EXCLUDED.billing_type,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP;

-- DALL-E 2 定价（如果使用）
INSERT INTO model_pricing (model_name, input_price, output_price, billing_type, is_active)
VALUES ('dall-e-2', 0.05, 0.10, 'image', true)
ON CONFLICT (model_name) DO UPDATE SET
    input_price = EXCLUDED.input_price,
    output_price = EXCLUDED.output_price,
    billing_type = EXCLUDED.billing_type,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP;

-- Stable Diffusion 定价（如果使用）
INSERT INTO model_pricing (model_name, input_price, output_price, billing_type, is_active)
VALUES ('stable-diffusion', 0.03, 0.06, 'image', true)
ON CONFLICT (model_name) DO UPDATE SET
    input_price = EXCLUDED.input_price,
    output_price = EXCLUDED.output_price,
    billing_type = EXCLUDED.billing_type,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP;

-- gpt-image-2 定价（根据实际模型名称）
INSERT INTO model_pricing (model_name, input_price, output_price, billing_type, is_active)
VALUES ('gpt-image-2', 0.08, 0.16, 'image', true)
ON CONFLICT (model_name) DO UPDATE SET
    input_price = EXCLUDED.input_price,
    output_price = EXCLUDED.output_price,
    billing_type = EXCLUDED.billing_type,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP;
