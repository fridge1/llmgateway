-- balances: add gift_balance and commission_balance columns
ALTER TABLE balances ADD COLUMN gift_balance DECIMAL(12,4) NOT NULL DEFAULT 0;
ALTER TABLE balances ADD COLUMN commission_balance DECIMAL(12,4) NOT NULL DEFAULT 0;

-- distributors: distributor metadata and stats
CREATE TABLE distributors (
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

-- distributor_users: users invited by distributors
CREATE TABLE distributor_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    distributor_id UUID NOT NULL REFERENCES distributors(id) ON DELETE CASCADE,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    registration_bonus_granted BOOLEAN NOT NULL DEFAULT FALSE,
    total_consumption DECIMAL(12,4) NOT NULL DEFAULT 0,
    total_paid_consumption DECIMAL(12,4) NOT NULL DEFAULT 0,
    total_commission_generated DECIMAL(12,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- commissions: individual commission records
CREATE TABLE commissions (
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

-- withdrawal_requests: commission withdrawal requests
CREATE TABLE withdrawal_requests (
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

-- users: add distributor_id foreign key
ALTER TABLE users ADD COLUMN distributor_id UUID REFERENCES distributors(id) ON DELETE SET NULL;

-- indexes
CREATE INDEX idx_distributors_user_id ON distributors(user_id);
CREATE INDEX idx_distributors_invite_code ON distributors(invite_code);
CREATE INDEX idx_distributor_users_distributor ON distributor_users(distributor_id);
CREATE INDEX idx_distributor_users_user ON distributor_users(user_id);
CREATE INDEX idx_commissions_distributor ON commissions(distributor_id, created_at DESC);
CREATE INDEX idx_commissions_user ON commissions(user_id);
CREATE INDEX idx_withdrawal_requests_distributor ON withdrawal_requests(distributor_id, created_at DESC);
CREATE INDEX idx_withdrawal_requests_status ON withdrawal_requests(status);
