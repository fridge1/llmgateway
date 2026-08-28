CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    order_no VARCHAR(64) UNIQUE NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    pay_method VARCHAR(20) NOT NULL DEFAULT 'alipay',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    pay_time TIMESTAMPTZ,
    callback_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expired_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_orders_user ON orders(user_id, created_at DESC);
CREATE INDEX idx_orders_no ON orders(order_no);
