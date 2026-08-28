ALTER TABLE tenant_pricing ADD COLUMN IF NOT EXISTS discount_rate NUMERIC(6,4);
COMMENT ON COLUMN tenant_pricing.discount_rate IS '折扣率(0-1]，非空时实际价=全局价×该值；为空时回退使用本表绝对单价(历史兼容)';
