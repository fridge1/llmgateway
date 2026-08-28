-- Allow multiple active subscriptions per user per brand (FIFO consumption model).
-- The old unique index enforced one active subscription per brand; drop it.
DROP INDEX IF EXISTS idx_user_subscriptions_active;

-- Non-unique index for efficient FIFO lookups (oldest first).
CREATE INDEX idx_user_subscriptions_user_brand_fifo
  ON user_subscriptions(user_id, brand, started_at ASC)
  WHERE status = 'active';
