CREATE TABLE ppt_tasks (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    phase VARCHAR(30) NOT NULL DEFAULT 'brief_analyst',

    -- User input
    topic TEXT NOT NULL,
    slide_count INTEGER NOT NULL DEFAULT 8,
    language VARCHAR(10) NOT NULL DEFAULT 'zh',
    theme VARCHAR(50) NOT NULL DEFAULT 'business-blue',
    audience VARCHAR(50) NOT NULL DEFAULT 'general',
    tone VARCHAR(50) NOT NULL DEFAULT 'professional',
    purpose VARCHAR(50) NOT NULL DEFAULT 'inform',

    -- Agent artifacts (JSONB)
    brief_document JSONB,
    story_arc JSONB,
    slide_blueprints JSONB,

    -- Final output
    presentation_json JSONB,

    -- Billing
    model VARCHAR(255) NOT NULL,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    cost DECIMAL(10, 6) NOT NULL DEFAULT 0,

    -- Error handling
    error_message TEXT,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_ppt_tasks_user_id ON ppt_tasks(user_id);
CREATE INDEX idx_ppt_tasks_status ON ppt_tasks(status);
