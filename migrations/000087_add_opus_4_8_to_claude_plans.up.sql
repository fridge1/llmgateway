INSERT INTO subscription_plan_models (plan_id, model_pattern)
SELECT p.id, 'claude-opus-4-8'
FROM subscription_plans p
WHERE p.name IN ('trial', 'pro', 'plus', 'premium', 'max');
