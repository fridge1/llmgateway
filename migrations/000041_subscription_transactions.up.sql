ALTER TABLE transactions
  ADD COLUMN subscription_id UUID REFERENCES user_subscriptions(id);

CREATE INDEX idx_transactions_user_type ON transactions(user_id, type, created_at DESC);
