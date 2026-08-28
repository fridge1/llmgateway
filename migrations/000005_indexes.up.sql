-- Composite index for billing stats queries that filter on type and order by created_at.
CREATE INDEX IF NOT EXISTS idx_transactions_user_type_time
    ON transactions(user_id, type, created_at DESC);
