-- migrations/000052_image_generations.down.sql
DROP INDEX IF EXISTS idx_image_generations_created_at;
DROP INDEX IF EXISTS idx_image_generations_user_id;
DROP INDEX IF EXISTS idx_image_generations_session_id;
DROP TABLE IF EXISTS image_generations;
