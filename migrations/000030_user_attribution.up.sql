-- Add attribution_type to users for mutual exclusion of referral vs distributor rewards.
ALTER TABLE users ADD COLUMN IF NOT EXISTS attribution_type VARCHAR(20) NOT NULL DEFAULT 'organic';

-- Backfill: users with distributor_users entry → 'distributor', users with referred_by → 'referral'.
-- Distributor takes priority if both exist (shouldn't happen, but be safe).
UPDATE users SET attribution_type = 'referral' WHERE referred_by IS NOT NULL;
UPDATE users SET attribution_type = 'distributor'
WHERE id IN (SELECT user_id FROM distributor_users);
