-- 每日签到：记录用户签到与连续天数，配合余额入账发放阶梯奖励。
CREATE TABLE IF NOT EXISTS daily_checkins (
    id           BIGSERIAL     PRIMARY KEY,
    user_id      UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkin_date DATE          NOT NULL,
    streak       INT           NOT NULL DEFAULT 1,
    reward_cny   NUMERIC(10,4) NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, checkin_date)
);
CREATE INDEX IF NOT EXISTS idx_daily_checkins_user ON daily_checkins (user_id, checkin_date DESC);
