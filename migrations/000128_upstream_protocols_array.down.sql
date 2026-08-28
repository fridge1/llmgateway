DROP INDEX IF EXISTS idx_tenant_model_upstreams_protocols;
DROP INDEX IF EXISTS idx_upstreams_protocols;
ALTER TABLE tenant_model_upstreams DROP COLUMN IF EXISTS protocols;
ALTER TABLE upstreams DROP COLUMN IF EXISTS protocols;
