INSERT INTO subscription_plan_models (plan_id, model_pattern)
SELECT p.id, 'claude-sonnet-5'
FROM subscription_plans p
WHERE p.name IN ('trial', 'pro', 'plus', 'max')
ON CONFLICT (plan_id, model_pattern) DO NOTHING;
