-- Restore the non-negative balance constraint.
-- NOTE: this will fail if any user currently has a negative balance.
ALTER TABLE balances ADD CONSTRAINT balance_non_negative CHECK (balance >= 0);
