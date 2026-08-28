-- Codex 代充服务模块

-- Codex 商品表
CREATE TABLE codex_products (
    id SERIAL PRIMARY KEY,
    sku VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price_cny DECIMAL(10,2) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_codex_products_status ON codex_products(status);
CREATE INDEX idx_codex_products_sort ON codex_products(sort_order);

-- 插入初始商品数据
INSERT INTO codex_products (sku, name, description, price_cny, sort_order, status) VALUES
    ('gpt-pro-20x', 'GPT Pro 20x', 'ChatGPT Pro 账号 20倍积分', 1350.00, 1, 'active'),
    ('gpt-pro-5x', 'GPT Pro 5x', 'ChatGPT Pro 账号 5倍积分', 730.00, 2, 'active'),
    ('gpt-plus', 'GPT Plus', 'ChatGPT Plus 账号', 140.00, 3, 'active');

-- Codex 订单表
CREATE TABLE codex_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_no VARCHAR(64) UNIQUE NOT NULL,
    product_id INT NOT NULL REFERENCES codex_products(id),

    -- 买家信息（游客或注册用户）
    user_id UUID REFERENCES users(id),
    guest_contact JSONB,

    -- 订单金额和支付
    amount DECIMAL(10,2) NOT NULL,
    pay_method VARCHAR(20) NOT NULL DEFAULT 'alipay',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    pay_time TIMESTAMPTZ,
    callback_data JSONB,

    -- 发货信息
    redemption_code TEXT,
    shipped_at TIMESTAMPTZ,
    shipped_by UUID REFERENCES users(id),

    -- 客服信息
    service_wechat TEXT DEFAULT 'codex-service-01',

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expired_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT codex_order_contact_required CHECK (
        user_id IS NOT NULL OR guest_contact IS NOT NULL
    )
);

CREATE INDEX idx_codex_orders_no ON codex_orders(order_no);
CREATE INDEX idx_codex_orders_user ON codex_orders(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_codex_orders_status ON codex_orders(status);
CREATE INDEX idx_codex_orders_created ON codex_orders(created_at DESC);
CREATE INDEX idx_codex_orders_guest_contact ON codex_orders USING gin(guest_contact) WHERE user_id IS NULL;

-- Codex 退款表
CREATE TABLE codex_refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    codex_order_no VARCHAR(64) NOT NULL REFERENCES codex_orders(order_no),
    amount DECIMAL(10,2) NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    out_request_no VARCHAR(100) UNIQUE NOT NULL,
    alipay_trade_no VARCHAR(100),
    operator_id UUID REFERENCES users(id),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_codex_refunds_order ON codex_refunds(codex_order_no);
CREATE INDEX idx_codex_refunds_status ON codex_refunds(status);
