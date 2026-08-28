-- Add soft delete support to ppt_tasks table
ALTER TABLE ppt_tasks ADD COLUMN deleted_at TIMESTAMPTZ;

-- Index for efficient queries on active tasks
CREATE INDEX idx_ppt_tasks_active ON ppt_tasks(user_id, created_at DESC) WHERE deleted_at IS NULL;
