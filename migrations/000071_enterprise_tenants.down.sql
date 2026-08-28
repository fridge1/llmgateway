DROP INDEX IF EXISTS idx_tenant_transactions_sub_user;
ALTER TABLE tenant_transactions DROP COLUMN IF EXISTS sub_user_username;
ALTER TABLE tenant_transactions DROP COLUMN IF EXISTS sub_user_id;

DROP TABLE IF EXISTS tenant_sub_user_keys;
DROP TABLE IF EXISTS tenant_sub_users;

DROP INDEX IF EXISTS idx_tenants_enterprise;
ALTER TABLE tenants DROP COLUMN IF EXISTS contact_email;
ALTER TABLE tenants DROP COLUMN IF EXISTS contact_phone;
ALTER TABLE tenants DROP COLUMN IF EXISTS created_by_admin;
ALTER TABLE tenants DROP COLUMN IF EXISTS is_enterprise;
