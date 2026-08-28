INSERT INTO subscription_plans (name, display_name, description, monthly_price_cny, quota_amount_cny, duration_days, sort_order)
VALUES
    ('openai-trial',   'OpenAI 体验版',   '体验周卡，适合新用户试用',     19.90,  29.00,    7,  10),
    ('openai-pro',     'OpenAI 开发者版', '入门订阅，适合轻度使用',       99.00,  129.00,  30,  11),
    ('openai-plus',    'OpenAI 专业版',   '专业订阅，适合日常开发',       299.00, 399.00,  30,  12),
    ('openai-premium', 'OpenAI 团队版',   '高级订阅，适合中高强度使用',   599.00, 799.00,  30,  13),
    ('openai-max',     'OpenAI 无限版',   '旗舰订阅，重度使用',           999.00, 1499.00, 30,  14);

INSERT INTO subscription_plan_models (plan_id, model_pattern)
SELECT p.id, m.model_pattern
FROM subscription_plans p
CROSS JOIN (VALUES
    ('gpt-5.4'),
    ('gpt-5.4-codex'),
    ('gpt-5.4-codex-high'),
    ('gpt-5.5')
) AS m(model_pattern)
WHERE p.name IN ('openai-trial', 'openai-pro', 'openai-plus', 'openai-premium', 'openai-max');
