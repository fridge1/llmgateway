-- Add api_key_id to transactions table for per-key billing tracking
ALTER TABLE transactions ADD COLUMN api_key_id UUID;

-- Index for efficient per-key transaction queries
CREATE INDEX idx_transactions_api_key ON transactions(api_key_id, created_at DESC)
  WHERE api_key_id IS NOT NULL;
