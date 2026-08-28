ALTER TABLE users
  DROP COLUMN IF EXISTS referral_reward_granted,
  DROP COLUMN IF EXISTS referred_by,
  DROP COLUMN IF EXISTS referral_code;
