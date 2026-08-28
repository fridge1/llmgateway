DROP INDEX IF EXISTS idx_invoice_requests_risk;
ALTER TABLE invoice_requests DROP COLUMN IF EXISTS risk_reasons;
ALTER TABLE invoice_requests DROP COLUMN IF EXISTS risk_level;
