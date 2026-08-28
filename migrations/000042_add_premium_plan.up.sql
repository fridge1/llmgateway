-- Bump Max sort_order from 3 to 4 to make room for Premium
UPDATE subscription_plans SET sort_order = 4, updated_at = NOW() WHERE name = 'max';

-- Insert Premium plan at sort_order = 3
INSERT INTO subscription_plans (name, display_name, description, monthly_price_cny, quota_amount_cny, sort_order)
VALUES ('premium', 'Premium', '高级订阅，适合中高强度使用', 599.00, 998.00, 3);

-- Associate the same 4 models with the new plan
INSERT INTO subscription_plan_models (plan_id, model_pattern)
SELECT p.id, m.model_pattern
FROM subscription_plans p
CROSS JOIN (VALUES
    ('claude-haiku-4-5-20251001'),
    ('claude-sonnet-4-6'),
    ('claude-opus-4-6'),
    ('claude-opus-4-7')
) AS m(model_pattern)
WHERE p.name = 'premium';
