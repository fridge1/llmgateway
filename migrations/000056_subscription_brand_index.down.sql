DROP INDEX idx_user_subscriptions_active;
CREATE UNIQUE INDEX idx_user_subscriptions_active ON user_subscriptions(user_id) WHERE status = 'active';
ALTER TABLE user_subscriptions DROP COLUMN brand;
