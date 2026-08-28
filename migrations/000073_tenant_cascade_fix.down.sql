-- Revert FK constraints to original (no cascade)
ALTER TABLE tenant_transactions DROP CONSTRAINT IF EXISTS tenant_transactions_sub_user_id_fkey;
ALTER TABLE tenant_transactions ADD CONSTRAINT tenant_transactions_sub_user_id_fkey
    FOREIGN KEY (sub_user_id) REFERENCES tenant_sub_users(id);

ALTER TABLE tenant_transactions DROP CONSTRAINT tenant_transactions_tenant_id_fkey;
ALTER TABLE tenant_transactions ADD CONSTRAINT tenant_transactions_tenant_id_fkey
    FOREIGN KEY (tenant_id) REFERENCES tenants(id);
