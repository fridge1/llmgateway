DROP INDEX IF EXISTS idx_image_sessions_share_key;
ALTER TABLE image_sessions DROP COLUMN IF EXISTS image_share_key_id;

DROP INDEX IF EXISTS idx_image_tasks_share_key;
ALTER TABLE image_tasks DROP COLUMN IF EXISTS image_share_key_id;

DROP INDEX IF EXISTS idx_image_share_keys_status;
DROP INDEX IF EXISTS idx_image_share_keys_owner;
DROP TABLE IF EXISTS image_share_keys;

ALTER TABLE users DROP COLUMN IF EXISTS image_share_enabled;
