-- 为 users 增加 last_active_at，用于留存运营（沉默用户识别、召回、DAU/WAU 统计）。
-- 可空，存量用户为 NULL（视为未知活跃时间），完全向后兼容。
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_users_last_active ON users (last_active_at);
