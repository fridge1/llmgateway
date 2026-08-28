-- Reverse: re-add referral/distributor schema (data is lost).

-- balances: re-add columns
ALTER TABLE balances ADD COLUMN gift_balance DECIMAL(12,4) NOT NULL DEFAULT 0;
ALTER TABLE balances ADD COLUMN commission_balance DECIMAL(12,4) NOT NULL DEFAULT 0;

-- users: re-add columns
ALTER TABLE users ADD COLUMN referral_code VARCHAR(8) UNIQUE;
ALTER TABLE users ADD COLUMN referred_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN referral_reward_granted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN attribution_type VARCHAR(20) NOT NULL DEFAULT 'organic';

-- Backfill referral_code for existing users
UPDATE users SET referral_code = UPPER(SUBSTRING(MD5(id::text) FROM 1 FOR 8)) WHERE referral_code IS NULL;
ALTER TABLE users ALTER COLUMN referral_code SET NOT NULL;

-- distributors
CREATE TABLE IF NOT EXISTS distributors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    commission_rate DECIMAL(5,4) NOT NULL DEFAULT 0.10,
    total_commission DECIMAL(12,4) NOT NULL DEFAULT 0,
    total_withdrawn DECIMAL(12,4) NOT NULL DEFAULT 0,
    invite_code VARCHAR(20) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN distributor_id UUID REFERENCES distributors(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS distributor_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    distributor_id UUID NOT NULL REFERENCES distributors(id) ON DELETE CASCADE,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    registration_bonus_granted BOOLEAN NOT NULL DEFAULT FALSE,
    first_recharge_commission_granted BOOLEAN NOT NULL DEFAULT FALSE,
    total_consumption DECIMAL(12,4) NOT NULL DEFAULT 0,
    total_paid_consumption DECIMAL(12,4) NOT NULL DEFAULT 0,
    total_commission_generated DECIMAL(12,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS commissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    distributor_id UUID NOT NULL REFERENCES distributors(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    transaction_id UUID,
    consumption_amount DECIMAL(12,4) NOT NULL,
    paid_amount DECIMAL(12,4) NOT NULL,
    commission_rate DECIMAL(5,4) NOT NULL,
    commission_amount DECIMAL(12,4) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'earning',
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS withdrawal_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    distributor_id UUID NOT NULL REFERENCES distributors(id) ON DELETE CASCADE,
    amount DECIMAL(12,4) NOT NULL,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    alipay_account VARCHAR(100),
    alipay_name VARCHAR(50),
    reject_reason TEXT,
    admin_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS registration_metadata (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    invite_code VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
