-- Remove referral and distributor features.
-- Merge gift_balance and commission_balance into balance before dropping.

-- balances: merge gift + commission into balance
UPDATE balances SET balance = balance + gift_balance + commission_balance
  WHERE gift_balance > 0 OR commission_balance > 0;
ALTER TABLE balances DROP COLUMN IF EXISTS gift_balance;
ALTER TABLE balances DROP COLUMN IF EXISTS commission_balance;

-- users: drop referral/distributor columns
ALTER TABLE users DROP COLUMN IF EXISTS referral_code;
ALTER TABLE users DROP COLUMN IF EXISTS referred_by;
ALTER TABLE users DROP COLUMN IF EXISTS referral_reward_granted;
ALTER TABLE users DROP COLUMN IF EXISTS distributor_id;
ALTER TABLE users DROP COLUMN IF EXISTS attribution_type;

-- drop tables in FK-dependency order
DROP TABLE IF EXISTS commissions;
DROP TABLE IF EXISTS withdrawal_requests;
DROP TABLE IF EXISTS distributor_users;
DROP TABLE IF EXISTS distributors;
DROP TABLE IF EXISTS registration_metadata;
