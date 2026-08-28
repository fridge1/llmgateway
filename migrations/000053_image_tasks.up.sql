CREATE TABLE image_tasks (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'generate',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    model VARCHAR(255) NOT NULL,
    prompt TEXT NOT NULL,
    size VARCHAR(50) NOT NULL DEFAULT '1024x1024',
    image_count INTEGER NOT NULL DEFAULT 1,
    input_images JSONB,
    input_mask BYTEA,
    result_urls JSONB,
    cost DECIMAL(10, 6) NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);
CREATE INDEX idx_image_tasks_user_id ON image_tasks(user_id);
CREATE INDEX idx_image_tasks_status ON image_tasks(status);
CREATE INDEX idx_image_tasks_created_at ON image_tasks(created_at DESC);
CREATE INDEX idx_image_tasks_pending ON image_tasks(status) WHERE status = 'pending';
