-- 通知去重表：避免同一用户同一类通知在同一天重复发送（如低余额预警、召回）。
-- kind 例如 'balance_low' / 'winback_7d' / 'winback_14d' / 'winback_30d'。
CREATE TABLE IF NOT EXISTS notification_dedup (
    user_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind     VARCHAR(50) NOT NULL,
    sent_on  DATE        NOT NULL DEFAULT CURRENT_DATE,
    PRIMARY KEY (user_id, kind, sent_on)
);
CREATE INDEX IF NOT EXISTS idx_notification_dedup_sent_on ON notification_dedup (sent_on);
