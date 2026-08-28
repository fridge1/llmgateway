DROP INDEX IF EXISTS idx_ppt_tasks_active;
ALTER TABLE ppt_tasks DROP COLUMN IF EXISTS deleted_at;
