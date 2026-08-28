ALTER TABLE user_subscriptions
  ADD COLUMN extra_quota_cny DECIMAL(10,2) NOT NULL DEFAULT 0;
