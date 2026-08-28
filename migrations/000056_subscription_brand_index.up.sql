-- Add brand column to support concurrent Claude + OpenAI subscriptions per user
ALTER TABLE user_subscriptions ADD COLUMN brand VARCHAR(20) NOT NULL DEFAULT 'claude';

-- Backfill: mark existing rows whose plan is an openai plan
UPDATE user_subscriptions us
SET brand = 'openai'
FROM subscription_plans sp
WHERE sp.id = us.plan_id AND sp.name LIKE 'openai-%';

-- Replace the old per-user unique constraint with a per-user-per-brand constraint
DROP INDEX idx_user_subscriptions_active;
CREATE UNIQUE INDEX idx_user_subscriptions_active ON user_subscriptions(user_id, brand) WHERE status = 'active';
