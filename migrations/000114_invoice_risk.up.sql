-- Invoice semi-automation: risk flag column for rule-based triage.
-- risk_level: '' (not evaluated) / 'auto_ok' (rules passed, batch-approvable)
--             / 'needs_review' (rule hit, manual attention required)
ALTER TABLE invoice_requests ADD COLUMN IF NOT EXISTS risk_level VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE invoice_requests ADD COLUMN IF NOT EXISTS risk_reasons TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_invoice_requests_risk ON invoice_requests (risk_level, created_at DESC);
