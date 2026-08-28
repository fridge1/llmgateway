ALTER TABLE tenants DROP COLUMN IF EXISTS moderation_enabled;
ALTER TABLE models DROP COLUMN IF EXISTS moderation_enabled;
DROP TABLE IF EXISTS moderation_hits;
DROP TABLE IF EXISTS moderation_keywords;
DROP TABLE IF EXISTS moderation_settings;
