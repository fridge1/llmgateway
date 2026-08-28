-- Add AUP acceptance tracking to users.
ALTER TABLE users ADD COLUMN IF NOT EXISTS aup_accepted_at TIMESTAMPTZ;
