INSERT INTO subscription_plans (name, display_name, description, monthly_price_cny, quota_amount_cny, sort_order)
VALUES
    ('pro',  'Pro',  '入门订阅，适合轻度使用', 99.00,  168.00,  1),
    ('plus', 'Plus', '专业订阅，适合日常开发',  299.00, 499.00,  2),
    ('max',  'Max',  '旗舰订阅，重度使用',      999.00, 1998.00, 3);

INSERT INTO subscription_plan_models (plan_id, model_pattern)
SELECT p.id, m.model_pattern
FROM subscription_plans p
CROSS JOIN (VALUES
    ('claude-haiku-4-5-20251001'),
    ('claude-sonnet-4-6'),
    ('claude-opus-4-6'),
    ('claude-opus-4-7')
) AS m(model_pattern);
