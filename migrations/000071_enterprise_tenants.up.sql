-- 扩展租户表，添加企业租户标识
ALTER TABLE tenants ADD COLUMN is_enterprise BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE tenants ADD COLUMN created_by_admin UUID REFERENCES users(id);
ALTER TABLE tenants ADD COLUMN contact_phone VARCHAR(20);
ALTER TABLE tenants ADD COLUMN contact_email VARCHAR(100);

-- 为企业租户添加索引
CREATE INDEX idx_tenants_enterprise ON tenants(is_enterprise) WHERE is_enterprise = true;

-- 租户子用户表（不计入平台用户统计）
CREATE TABLE tenant_sub_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username VARCHAR(50) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    quota_limit NUMERIC(12,4) DEFAULT NULL,  -- NULL 表示无限制
    quota_used NUMERIC(12,4) NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, username)
);

CREATE INDEX idx_tenant_sub_users_tenant ON tenant_sub_users(tenant_id);
CREATE INDEX idx_tenant_sub_users_status ON tenant_sub_users(status);

-- 子用户 API Key
CREATE TABLE tenant_sub_user_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sub_user_id UUID NOT NULL REFERENCES tenant_sub_users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL DEFAULT '',
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    key_prefix VARCHAR(12) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tenant_sub_user_keys_hash ON tenant_sub_user_keys(key_hash);
CREATE INDEX idx_tenant_sub_user_keys_sub_user ON tenant_sub_user_keys(sub_user_id);

-- 扩展租户交易记录，添加子用户信息
ALTER TABLE tenant_transactions ADD COLUMN sub_user_id UUID REFERENCES tenant_sub_users(id);
ALTER TABLE tenant_transactions ADD COLUMN sub_user_username VARCHAR(50);

CREATE INDEX idx_tenant_transactions_sub_user ON tenant_transactions(sub_user_id, created_at DESC)
    WHERE sub_user_id IS NOT NULL;
