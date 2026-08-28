DROP INDEX IF EXISTS idx_users_last_active;
ALTER TABLE users DROP COLUMN IF EXISTS last_active_at;
