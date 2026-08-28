DELETE FROM subscription_plan_models WHERE plan_id IN (
    SELECT id FROM subscription_plans WHERE name LIKE 'openai-%'
);
DELETE FROM subscription_plans WHERE name LIKE 'openai-%';
