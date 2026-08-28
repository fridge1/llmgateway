-- Add eligibility_type column to lottery_events table
-- Allows lottery events to be based on either spending or recharge amount

ALTER TABLE lottery_events
ADD COLUMN eligibility_type VARCHAR(20) NOT NULL DEFAULT 'spend';

-- Add check constraint to ensure valid values
ALTER TABLE lottery_events
ADD CONSTRAINT chk_lottery_eligibility_type
CHECK (eligibility_type IN ('spend', 'recharge'));

-- Create index for filtering by eligibility type
CREATE INDEX idx_lottery_events_eligibility_type ON lottery_events (eligibility_type);
