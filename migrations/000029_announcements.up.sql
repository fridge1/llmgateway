CREATE TABLE announcements (
    id           BIGSERIAL PRIMARY KEY,
    title        VARCHAR(255) NOT NULL,
    content      TEXT         NOT NULL DEFAULT '',
    status       VARCHAR(20)  NOT NULL DEFAULT 'draft',
    priority     VARCHAR(20)  NOT NULL DEFAULT 'normal',
    display_mode VARCHAR(20)  NOT NULL DEFAULT 'banner',
    created_by   UUID         REFERENCES users(id) ON DELETE SET NULL,
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_announcements_status_published ON announcements (status, published_at DESC);
