-- 1. 订阅计划定义
CREATE TABLE subscription_plans (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    monthly_price_cny DECIMAL(10,2) NOT NULL,
    quota_amount_cny  DECIMAL(10,2) NOT NULL,
    sort_order  INT NOT NULL DEFAULT 0,
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. 计划覆盖的模型列表
CREATE TABLE subscription_plan_models (
    id              SERIAL PRIMARY KEY,
    plan_id         INT NOT NULL REFERENCES subscription_plans(id) ON DELETE CASCADE,
    model_pattern   VARCHAR(200) NOT NULL,
    UNIQUE(plan_id, model_pattern)
);

-- 3. 用户订阅
CREATE TABLE user_subscriptions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    plan_id     INT NOT NULL REFERENCES subscription_plans(id),
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    auto_renew  BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_user_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX idx_user_subscriptions_expires ON user_subscriptions(expires_at) WHERE status = 'active';
CREATE UNIQUE INDEX idx_user_subscriptions_active ON user_subscriptions(user_id) WHERE status = 'active';

-- 4. 订阅用量跟踪
CREATE TABLE subscription_usage (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL,
    subscription_id UUID NOT NULL REFERENCES user_subscriptions(id),
    model_name      VARCHAR(200) NOT NULL,
    period          DATE NOT NULL,
    input_tokens_used            BIGINT NOT NULL DEFAULT 0,
    output_tokens_used           BIGINT NOT NULL DEFAULT 0,
    cache_read_tokens_used       BIGINT NOT NULL DEFAULT 0,
    cache_creation_tokens_used   BIGINT NOT NULL DEFAULT 0,
    amount_used     DECIMAL(10,4) NOT NULL DEFAULT 0,
    request_count   INT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(subscription_id, model_name, period)
);
CREATE INDEX idx_subscription_usage_lookup ON subscription_usage(user_id, period);

-- 汇总视图
CREATE VIEW subscription_usage_summary AS
SELECT subscription_id, period,
       SUM(amount_used) AS total_amount_used,
       SUM(input_tokens_used) AS total_input_tokens,
       SUM(output_tokens_used) AS total_output_tokens,
       SUM(cache_read_tokens_used) AS total_cache_read_tokens,
       SUM(cache_creation_tokens_used) AS total_cache_creation_tokens,
       SUM(request_count) AS total_requests
FROM subscription_usage
GROUP BY subscription_id, period;

-- 5. 订阅订单
CREATE TABLE subscription_orders (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    plan_id     INT NOT NULL REFERENCES subscription_plans(id),
    amount_cny  DECIMAL(10,2) NOT NULL,
    type        VARCHAR(20) NOT NULL DEFAULT 'new',
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    payment_method VARCHAR(50),
    payment_id  VARCHAR(200),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at     TIMESTAMPTZ
);
