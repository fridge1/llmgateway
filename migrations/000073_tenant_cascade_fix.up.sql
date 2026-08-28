-- Fix FK constraints for tenant deletion cascade
-- 1) tenant_transactions.sub_user_id → ON DELETE SET NULL
ALTER TABLE tenant_transactions DROP CONSTRAINT IF EXISTS tenant_transactions_sub_user_id_fkey;
ALTER TABLE tenant_transactions ADD CONSTRAINT tenant_transactions_sub_user_id_fkey
    FOREIGN KEY (sub_user_id) REFERENCES tenant_sub_users(id) ON DELETE SET NULL;

-- 2) tenant_transactions.tenant_id → ON DELETE CASCADE
ALTER TABLE tenant_transactions DROP CONSTRAINT tenant_transactions_tenant_id_fkey;
ALTER TABLE tenant_transactions ADD CONSTRAINT tenant_transactions_tenant_id_fkey
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
