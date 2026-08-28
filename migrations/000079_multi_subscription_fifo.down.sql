DROP INDEX IF EXISTS idx_user_subscriptions_user_brand_fifo;

-- Restore the unique constraint (one active subscription per user per brand).
CREATE UNIQUE INDEX idx_user_subscriptions_active
  ON user_subscriptions(user_id, brand)
  WHERE status = 'active';
