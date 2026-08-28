-- Alerting: configurable ops alert rules and fired alert events.
CREATE TABLE IF NOT EXISTS alert_rules (
    id SERIAL PRIMARY KEY,
    metric VARCHAR(64) NOT NULL UNIQUE,
    display_name VARCHAR(128) NOT NULL,
    threshold BIGINT NOT NULL DEFAULT 1,
    cooldown_seconds INT NOT NULL DEFAULT 3600,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed default rules. Threshold is the per-check-interval delta that triggers.
INSERT INTO alert_rules (metric, display_name, threshold, cooldown_seconds, enabled) VALUES
    ('circuit_opened',         '上游熔断触发',       1,  1800, TRUE),
    ('billing_jobs_dropped',   '计费结算任务丢弃',   1,  1800, TRUE),
    ('billing_queue_overflow', '计费队列溢出',       10, 3600, TRUE),
    ('upstream_failures',      '上游请求失败激增',   50, 3600, TRUE),
    ('db_health',              '数据库健康检查失败', 1,  900,  TRUE)
ON CONFLICT (metric) DO NOTHING;

CREATE TABLE IF NOT EXISTS alert_events (
    id BIGSERIAL PRIMARY KEY,
    metric VARCHAR(64) NOT NULL,
    message TEXT NOT NULL,
    value BIGINT NOT NULL,
    threshold BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_alert_events_created ON alert_events (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_events_metric_created ON alert_events (metric, created_at DESC);
