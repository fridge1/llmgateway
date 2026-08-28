-- Rollback eligibility_type column from lottery_events table

DROP INDEX IF EXISTS idx_lottery_events_eligibility_type;

ALTER TABLE lottery_events DROP CONSTRAINT IF EXISTS chk_lottery_eligibility_type;

ALTER TABLE lottery_events DROP COLUMN IF EXISTS eligibility_type;
