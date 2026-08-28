ALTER TABLE lottery_events
  DROP COLUMN IF EXISTS drawn_by,
  DROP COLUMN IF EXISTS drawn_at;
