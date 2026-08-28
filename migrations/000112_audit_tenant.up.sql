-- Tenant-scoped audit: allow tagging audit rows with a tenant and let tenant
-- owners view their own trail. Backward compatible (new nullable column).
ALTER TABLE admin_audit_logs ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_tenant ON admin_audit_logs (tenant_id, created_at DESC);
