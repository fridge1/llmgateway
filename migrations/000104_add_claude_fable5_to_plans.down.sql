DELETE FROM subscription_plan_models
WHERE model_pattern = 'claude-fable-5'
  AND plan_id IN (SELECT id FROM subscription_plans WHERE name IN ('trial', 'pro', 'plus', 'max'));
