DELETE FROM subscription_plan_models WHERE plan_id = (SELECT id FROM subscription_plans WHERE name = 'premium');
DELETE FROM subscription_plans WHERE name = 'premium';
UPDATE subscription_plans SET sort_order = 3, updated_at = NOW() WHERE name = 'max';
