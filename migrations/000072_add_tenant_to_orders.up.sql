-- Add tenant_id to orders table to support tenant-based recharges
ALTER TABLE orders ADD COLUMN tenant_id UUID REFERENCES tenants(id);

-- Create index for efficient tenant order queries
CREATE INDEX idx_orders_tenant ON orders(tenant_id);
