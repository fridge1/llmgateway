-- Refunds: admin-initiated Alipay refunds with balance clawback.
CREATE TABLE IF NOT EXISTS refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_no VARCHAR(64) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    amount DECIMAL(12,4) NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',  -- pending / success / failed
    out_request_no VARCHAR(64) NOT NULL UNIQUE,     -- idempotency key sent to Alipay
    alipay_trade_no VARCHAR(64),
    operator_id UUID,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_refunds_order ON refunds (order_no);
CREATE INDEX IF NOT EXISTS idx_refunds_user ON refunds (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_refunds_created ON refunds (created_at DESC);
