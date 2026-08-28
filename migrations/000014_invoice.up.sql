-- migrations/000014_invoice.up.sql

-- 发票抬头
CREATE TABLE invoice_titles (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL DEFAULT 'personal',
    title_name VARCHAR(200) NOT NULL,
    tax_number VARCHAR(30) DEFAULT '',
    bank_name VARCHAR(100) DEFAULT '',
    bank_account VARCHAR(50) DEFAULT '',
    address VARCHAR(200) DEFAULT '',
    phone VARCHAR(20) DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_invoice_titles_user ON invoice_titles(user_id);
CREATE INDEX idx_invoice_titles_user_default ON invoice_titles(user_id, is_default);

-- 开票申请
CREATE TABLE invoice_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    title_id BIGINT NOT NULL REFERENCES invoice_titles(id),
    invoice_type VARCHAR(20) NOT NULL DEFAULT 'normal',
    total_amount DECIMAL(12,2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    remark TEXT DEFAULT '',
    reject_reason TEXT DEFAULT '',
    invoice_file_path VARCHAR(500) DEFAULT '',
    invoice_number VARCHAR(50) DEFAULT '',
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_invoice_requests_user ON invoice_requests(user_id, created_at DESC);
CREATE INDEX idx_invoice_requests_status ON invoice_requests(status);

-- 申请关联订单
CREATE TABLE invoice_request_orders (
    id BIGSERIAL PRIMARY KEY,
    request_id BIGINT NOT NULL REFERENCES invoice_requests(id),
    order_id UUID NOT NULL REFERENCES orders(id),
    amount DECIMAL(12,2) NOT NULL
);
CREATE INDEX idx_invoice_request_orders_order ON invoice_request_orders(order_id);
CREATE UNIQUE INDEX idx_invoice_request_orders_unique ON invoice_request_orders(order_id, request_id);
