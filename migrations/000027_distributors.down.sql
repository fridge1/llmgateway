DROP INDEX IF EXISTS idx_withdrawal_requests_status;
DROP INDEX IF EXISTS idx_withdrawal_requests_distributor;
DROP INDEX IF EXISTS idx_commissions_user;
DROP INDEX IF EXISTS idx_commissions_distributor;
DROP INDEX IF EXISTS idx_distributor_users_user;
DROP INDEX IF EXISTS idx_distributor_users_distributor;
DROP INDEX IF EXISTS idx_distributors_invite_code;
DROP INDEX IF EXISTS idx_distributors_user_id;

ALTER TABLE users DROP COLUMN IF EXISTS distributor_id;

DROP TABLE IF EXISTS withdrawal_requests;
DROP TABLE IF EXISTS commissions;
DROP TABLE IF EXISTS distributor_users;
DROP TABLE IF EXISTS distributors;

ALTER TABLE balances DROP COLUMN IF EXISTS commission_balance;
ALTER TABLE balances DROP COLUMN IF EXISTS gift_balance;
