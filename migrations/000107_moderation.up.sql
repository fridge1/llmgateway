-- Content moderation: keyword rules, hit records, and per-model/tenant switches.
CREATE TABLE IF NOT EXISTS moderation_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    -- enforce_all: apply to every request regardless of per-model/tenant flags.
    enforce_all BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO moderation_settings (id, enabled, enforce_all) VALUES (1, FALSE, TRUE)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS moderation_keywords (
    id SERIAL PRIMARY KEY,
    keyword TEXT NOT NULL UNIQUE,
    category VARCHAR(32) NOT NULL DEFAULT 'general',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS moderation_hits (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID,
    tenant_id UUID,
    model VARCHAR(128) NOT NULL,
    matched_rule TEXT NOT NULL,
    snippet TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_moderation_hits_created ON moderation_hits (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_moderation_hits_user ON moderation_hits (user_id, created_at DESC);

-- Per-model / per-tenant opt-in switches (used when enforce_all is off).
ALTER TABLE models ADD COLUMN IF NOT EXISTS moderation_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS moderation_enabled BOOLEAN NOT NULL DEFAULT FALSE;
