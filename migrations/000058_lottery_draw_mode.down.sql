DROP INDEX IF EXISTS idx_lottery_records_event_user;
ALTER TABLE lottery_records DROP COLUMN won;
ALTER TABLE lottery_records ALTER COLUMN prize_id SET NOT NULL;
ALTER TABLE lottery_events DROP COLUMN draw_mode;
