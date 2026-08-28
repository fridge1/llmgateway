-- 为 users 表添加邮箱认证支持

-- 1. 添加 email 字段
ALTER TABLE users ADD COLUMN email VARCHAR(255);

-- 2. 添加邮箱验证状态字段
ALTER TABLE users ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- 3. 添加邮箱验证时间字段
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;

-- 4. 允许 phone 为 NULL（向后兼容，新用户可以只用邮箱注册）
ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;

-- 5. 添加约束：phone 和 email 至少有一个
ALTER TABLE users ADD CONSTRAINT users_phone_or_email_required
  CHECK (phone IS NOT NULL OR email IS NOT NULL);

-- 6. 创建邮箱唯一索引（部分索引，仅当 email 非 NULL 时）
CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
