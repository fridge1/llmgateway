-- migrations/000052_image_generations.up.sql
CREATE TABLE image_generations (
    id SERIAL PRIMARY KEY,
    session_id INTEGER NOT NULL REFERENCES image_sessions(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL,
    prompt TEXT NOT NULL,
    size VARCHAR(50) NOT NULL,
    image_count INTEGER NOT NULL,
    image_urls JSONB NOT NULL,
    cost DECIMAL(10, 6) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_image_generations_session_id ON image_generations(session_id);
CREATE INDEX idx_image_generations_user_id ON image_generations(user_id);
CREATE INDEX idx_image_generations_created_at ON image_generations(created_at DESC);
