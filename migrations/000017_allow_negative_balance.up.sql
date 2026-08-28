-- Allow balance to go negative so post-request billing always succeeds.
-- The gateway checks balance > 0 before each request; if a single request
-- pushes balance below zero, subsequent requests will be rejected.
ALTER TABLE balances DROP CONSTRAINT IF EXISTS balance_non_negative;
