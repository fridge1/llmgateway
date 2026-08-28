-- migrations/000051_image_sessions.down.sql
DROP INDEX IF EXISTS idx_image_sessions_created_at;
DROP INDEX IF EXISTS idx_image_sessions_user_id;
DROP TABLE IF EXISTS image_sessions;
