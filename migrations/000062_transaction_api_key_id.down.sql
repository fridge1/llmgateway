-- Remove api_key_id from transactions table
DROP INDEX IF EXISTS idx_transactions_api_key;
ALTER TABLE transactions DROP COLUMN IF EXISTS api_key_id;
