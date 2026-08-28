ALTER TABLE image_tasks DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE image_tasks DROP COLUMN IF EXISTS tenant_key_id;
ALTER TABLE image_tasks DROP COLUMN IF EXISTS sub_user_id;
ALTER TABLE image_tasks DROP COLUMN IF EXISTS sub_user_key_id;
ALTER TABLE image_tasks DROP COLUMN IF EXISTS api_key_id;
