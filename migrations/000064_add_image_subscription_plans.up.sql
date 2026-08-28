-- 统一 gpt-image-2 定价，不区分分辨率
UPDATE model_pricing SET output_price = input_price, updated_at = NOW()
WHERE model_name = 'gpt-image-2';

INSERT INTO subscription_plans (name, display_name, description, monthly_price_cny, quota_amount_cny, duration_days, sort_order)
VALUES
    ('image-basic', '图片基础版', '每月400张图片生成', 100.00, 32.00, 30, 20),
    ('image-pro',   '图片专业版', '每月1000张图片生成', 200.00, 80.00, 30, 21),
    ('image-max',   '图片旗舰版', '每月2000张图片生成', 300.00, 160.00, 30, 22);

INSERT INTO subscription_plan_models (plan_id, model_pattern)
SELECT p.id, 'gpt-image-2'
FROM subscription_plans p
WHERE p.name IN ('image-basic', 'image-pro', 'image-max');

-- 确保描述使用"张"而非"次"
UPDATE subscription_plans SET description = '每月400张图片生成' WHERE name = 'image-basic';
UPDATE subscription_plans SET description = '每月1000张图片生成' WHERE name = 'image-pro';
UPDATE subscription_plans SET description = '每月2000张图片生成' WHERE name = 'image-max';
