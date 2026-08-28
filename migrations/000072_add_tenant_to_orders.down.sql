-- Remove tenant_id index
DROP INDEX IF EXISTS idx_orders_tenant;

-- Remove tenant_id column from orders table
ALTER TABLE orders DROP COLUMN IF EXISTS tenant_id;
