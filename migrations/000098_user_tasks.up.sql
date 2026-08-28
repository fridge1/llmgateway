-- 成长任务体系：任务定义 + 用户进度，激励新用户走完激活漏斗。
CREATE TABLE IF NOT EXISTS task_definitions (
    id                   SERIAL       PRIMARY KEY,
    code                 VARCHAR(50)  UNIQUE NOT NULL,
    title                VARCHAR(100) NOT NULL,
    description          TEXT         NOT NULL DEFAULT '',
    reward_cny           NUMERIC(10,4) NOT NULL DEFAULT 0,
    reward_lottery_draws INT          NOT NULL DEFAULT 0,
    sort_order           INT          NOT NULL DEFAULT 0,
    is_active            BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_task_progress (
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_code    VARCHAR(50) NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending | completed | claimed
    completed_at TIMESTAMPTZ,
    claimed_at   TIMESTAMPTZ,
    PRIMARY KEY (user_id, task_code)
);
CREATE INDEX IF NOT EXISTS idx_user_task_progress_user ON user_task_progress (user_id);

-- Seed the initial activation tasks.
INSERT INTO task_definitions (code, title, description, reward_cny, sort_order) VALUES
    ('first_api_call',   '完成首次 API 调用',   '使用任意模型成功发起一次调用',        1.0, 1),
    ('first_recharge',   '完成首次充值',         '为账户充值任意金额',                  2.0, 2),
    ('daily_spend_1',    '单日消费突破 ¥1',      '在任意一天内累计消费满 ¥1',           1.0, 3),
    ('try_image',        '体验图像生成',         '使用图像生成功能产出一张图',          1.0, 4),
    ('try_ppt',          '体验 PPT 生成',        '使用 PPT 生成功能创建一份演示',       1.0, 5)
ON CONFLICT (code) DO NOTHING;
