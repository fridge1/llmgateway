-- lottery_events: add draw_mode column (instant = realtime, post = draw after event ends)
ALTER TABLE lottery_events ADD COLUMN draw_mode VARCHAR(20) NOT NULL DEFAULT 'instant';

-- lottery_records: prize_id nullable (NULL = joined but not yet drawn, or lost)
ALTER TABLE lottery_records ALTER COLUMN prize_id DROP NOT NULL;

-- lottery_records: won flag (default true for backward compat with existing instant-mode records)
ALTER TABLE lottery_records ADD COLUMN won BOOLEAN NOT NULL DEFAULT true;

-- unique constraint needed for ON CONFLICT in auto-enroll
CREATE UNIQUE INDEX IF NOT EXISTS idx_lottery_records_event_user ON lottery_records (event_id, user_id);
