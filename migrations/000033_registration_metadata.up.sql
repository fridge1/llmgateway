-- Registration metadata for anti-fraud detection.
CREATE TABLE IF NOT EXISTS registration_metadata (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address  TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    invite_code TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_registration_metadata_ip ON registration_metadata (ip_address);
CREATE INDEX idx_registration_metadata_created ON registration_metadata (created_at DESC);
CREATE INDEX idx_registration_metadata_user ON registration_metadata (user_id);
