CREATE TABLE balances (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    balance DECIMAL(12,4) NOT NULL DEFAULT 0,
    frozen DECIMAL(12,4) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT balance_non_negative CHECK (balance >= 0),
    CONSTRAINT frozen_non_negative CHECK (frozen >= 0)
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL,
    amount DECIMAL(12,4) NOT NULL,
    balance_after DECIMAL(12,4) NOT NULL,
    model VARCHAR(100),
    description TEXT,
    request_id VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_transactions_user_time ON transactions(user_id, created_at DESC);

CREATE TABLE model_pricing (
    id SERIAL PRIMARY KEY,
    model_name VARCHAR(100) UNIQUE NOT NULL,
    input_price DECIMAL(10,6) NOT NULL,
    output_price DECIMAL(10,6) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auto-create balance row when a user is inserted.
CREATE OR REPLACE FUNCTION create_balance_for_user()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO balances (user_id) VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_create_balance_after_user_insert
    AFTER INSERT ON users
    FOR EACH ROW
    EXECUTE FUNCTION create_balance_for_user();

-- Backfill balances for existing users that don't have one yet.
INSERT INTO balances (user_id)
SELECT id FROM users WHERE id NOT IN (SELECT user_id FROM balances);
