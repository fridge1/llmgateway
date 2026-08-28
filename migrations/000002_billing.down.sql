DROP TRIGGER IF EXISTS trg_create_balance_after_user_insert ON users;
DROP FUNCTION IF EXISTS create_balance_for_user();
DROP TABLE IF EXISTS model_pricing;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS balances;
