-- migrations/000051_image_sessions.up.sql
CREATE TABLE image_sessions (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_image_sessions_user_id ON image_sessions(user_id);
CREATE INDEX idx_image_sessions_created_at ON image_sessions(created_at DESC);
