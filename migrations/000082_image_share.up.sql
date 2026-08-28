ALTER TABLE users ADD COLUMN IF NOT EXISTS image_share_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS image_share_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash        VARCHAR(64) NOT NULL UNIQUE,
    key_prefix      VARCHAR(20) NOT NULL,
    name            VARCHAR(64) NOT NULL DEFAULT '',
    quota_total     INTEGER NOT NULL CHECK (quota_total >= 0),
    quota_used      INTEGER NOT NULL DEFAULT 0 CHECK (quota_used >= 0),
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_image_share_keys_owner ON image_share_keys(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_image_share_keys_status ON image_share_keys(status);

ALTER TABLE image_tasks ADD COLUMN IF NOT EXISTS image_share_key_id UUID;
CREATE INDEX IF NOT EXISTS idx_image_tasks_share_key ON image_tasks(image_share_key_id) WHERE image_share_key_id IS NOT NULL;

ALTER TABLE image_sessions ADD COLUMN IF NOT EXISTS image_share_key_id UUID;
CREATE INDEX IF NOT EXISTS idx_image_sessions_share_key ON image_sessions(image_share_key_id) WHERE image_share_key_id IS NOT NULL;
