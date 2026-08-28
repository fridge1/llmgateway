DROP INDEX IF EXISTS idx_admin_audit_logs_tenant;
ALTER TABLE admin_audit_logs DROP COLUMN IF EXISTS tenant_id;
