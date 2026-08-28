-- 回滚邮箱认证功能

-- 1. 删除邮箱唯一索引
DROP INDEX IF EXISTS idx_users_email;

-- 2. 删除约束
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_or_email_required;

-- 3. 恢复 phone 字段为 NOT NULL（注意：如果已有仅邮箱用户，此操作会失败）
ALTER TABLE users ALTER COLUMN phone SET NOT NULL;

-- 4. 删除邮箱验证时间字段
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;

-- 5. 删除邮箱验证状态字段
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;

-- 6. 删除 email 字段
ALTER TABLE users DROP COLUMN IF EXISTS email;
