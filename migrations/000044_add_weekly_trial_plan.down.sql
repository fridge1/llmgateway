-- Remove trial plan model associations and plan
DELETE FROM subscription_plan_models WHERE plan_id = (SELECT id FROM subscription_plans WHERE name = 'trial');
DELETE FROM subscription_plans WHERE name = 'trial';

-- Drop duration_days column
ALTER TABLE subscription_plans DROP COLUMN IF EXISTS duration_days;
