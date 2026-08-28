DROP INDEX IF EXISTS idx_transactions_user_type;
ALTER TABLE transactions DROP COLUMN IF EXISTS subscription_id;
