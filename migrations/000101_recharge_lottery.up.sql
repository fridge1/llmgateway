-- 充值累积抽奖活动配置
CREATE TABLE recharge_lottery (
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(100) NOT NULL DEFAULT '充值幸运奖',
    status        VARCHAR(20)  NOT NULL DEFAULT 'active', -- active | paused
    trigger_every INT          NOT NULL DEFAULT 10,       -- 每 N 笔触发一次开奖
    total_rounds  INT          NOT NULL DEFAULT 0,        -- 已完成开奖轮数
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 充值参与记录（每笔充值订单一条，order_no 保证幂等）
CREATE TABLE recharge_lottery_entries (
    id          BIGSERIAL    PRIMARY KEY,
    lottery_id  INT          NOT NULL REFERENCES recharge_lottery(id),
    user_id     UUID         NOT NULL REFERENCES users(id),
    order_no    VARCHAR(64)  NOT NULL UNIQUE,  -- 与 orders.order_no 对应，防重复参与
    amount      NUMERIC(10,4) NOT NULL,
    round_no    INT,                            -- NULL=待开奖；N=已参与第 N 轮
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_rle_pending ON recharge_lottery_entries(lottery_id) WHERE round_no IS NULL;

-- 开奖结果记录
CREATE TABLE recharge_lottery_rounds (
    id                BIGSERIAL    PRIMARY KEY,
    lottery_id        INT          NOT NULL REFERENCES recharge_lottery(id),
    round_no          INT          NOT NULL,
    winner_user_id    UUID         NOT NULL REFERENCES users(id),
    winner_amount     NUMERIC(10,4) NOT NULL,   -- 中奖者充值金额，即赠送金额
    winner_order_no   VARCHAR(64)  NOT NULL,    -- 中奖的那笔订单号
    participant_count INT          NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(lottery_id, round_no)
);
