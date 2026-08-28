-- Add duration_days column (default 30 for existing monthly plans)
ALTER TABLE subscription_plans ADD COLUMN duration_days INT NOT NULL DEFAULT 30;

-- Insert Trial weekly plan at sort_order 0 (before Pro)
INSERT INTO subscription_plans (name, display_name, description, monthly_price_cny, quota_amount_cny, sort_order, duration_days)
VALUES ('trial', 'Trial', '体验周卡，适合新用户试用', 19.90, 29.00, 0, 7);

-- Associate the same 4 models with the trial plan
INSERT INTO subscription_plan_models (plan_id, model_pattern)
SELECT p.id, m.model_pattern
FROM subscription_plans p
CROSS JOIN (VALUES
    ('claude-haiku-4-5-20251001'),
    ('claude-sonnet-4-6'),
    ('claude-opus-4-6'),
    ('claude-opus-4-7')
) AS m(model_pattern)
WHERE p.name = 'trial';
