-- Add soft delete support to api_keys table
ALTER TABLE api_keys ADD COLUMN deleted_at TIMESTAMPTZ;

-- Index for efficient queries on active keys
CREATE INDEX idx_api_keys_active ON api_keys(user_id) WHERE deleted_at IS NULL;
