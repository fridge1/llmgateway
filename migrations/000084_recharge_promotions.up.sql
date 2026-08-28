CREATE TABLE recharge_promotions (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    starts_at       TIMESTAMPTZ NOT NULL,
    ends_at         TIMESTAMPTZ NOT NULL,
    bonus_ratio     DECIMAL(5,4) NOT NULL CHECK (bonus_ratio > 0 AND bonus_ratio <= 1),
    min_recharge_amount DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (min_recharge_amount >= 0),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (ends_at > starts_at)
);

CREATE INDEX idx_recharge_promotions_active_window
    ON recharge_promotions (is_active, starts_at, ends_at);
