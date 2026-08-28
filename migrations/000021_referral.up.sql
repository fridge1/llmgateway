-- Add referral_code (unique invite code per user) and referred_by (who invited this user)
ALTER TABLE users
  ADD COLUMN referral_code VARCHAR(8) UNIQUE,
  ADD COLUMN referred_by UUID REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN referral_reward_granted BOOLEAN NOT NULL DEFAULT FALSE;

-- Backfill existing users with a random 8-char code
UPDATE users SET referral_code = UPPER(SUBSTRING(MD5(id::text) FROM 1 FOR 8));

-- Make referral_code NOT NULL after backfill
ALTER TABLE users ALTER COLUMN referral_code SET NOT NULL;
