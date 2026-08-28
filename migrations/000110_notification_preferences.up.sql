-- Per-user notification channel preferences.
-- Default behaviour when no row exists: in-app on, SMS off.
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(32) NOT NULL,   -- balance_low / subscription_expiry / ticket / ops_alert
    channel VARCHAR(16) NOT NULL,      -- sms (in-app is always on)
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, event_type, channel)
);
