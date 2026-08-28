-- 轻量邀请奖励：复活邀请码 + 被邀请人关系（参考已删除的历史迁移 000021）。
-- 仅做邀请裂变，不重建分销/提现/佣金（那是 000039 删除的复杂部分）。
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS referral_code VARCHAR(8) UNIQUE,
  ADD COLUMN IF NOT EXISTS referred_by UUID REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS referral_reward_granted BOOLEAN NOT NULL DEFAULT FALSE;

-- 为存量用户回填邀请码（取 id 的 MD5 前 8 位，大写）。
UPDATE users SET referral_code = UPPER(SUBSTRING(MD5(id::text) FROM 1 FOR 8))
  WHERE referral_code IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_referred_by ON users (referred_by);
