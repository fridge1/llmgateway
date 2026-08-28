-- Remove soft delete support from api_keys table
DROP INDEX IF EXISTS idx_api_keys_active;
ALTER TABLE api_keys DROP COLUMN IF EXISTS deleted_at;
