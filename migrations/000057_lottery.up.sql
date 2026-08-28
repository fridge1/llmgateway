-- Lottery system: events, prizes, and draw records

CREATE TABLE lottery_events (
    id              BIGSERIAL    PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    description     TEXT         NOT NULL DEFAULT '',
    status          VARCHAR(20)  NOT NULL DEFAULT 'draft',
    start_time      TIMESTAMPTZ  NOT NULL,
    end_time        TIMESTAMPTZ  NOT NULL,
    min_spend_cny   NUMERIC(10,4) NOT NULL DEFAULT 0,
    created_by      UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lottery_events_status_time ON lottery_events (status, start_time, end_time);

CREATE TABLE lottery_prizes (
    id              BIGSERIAL    PRIMARY KEY,
    event_id        BIGINT       NOT NULL REFERENCES lottery_events(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    description     TEXT         NOT NULL DEFAULT '',
    weight          INT          NOT NULL DEFAULT 100,
    total_stock     INT          NOT NULL DEFAULT 0,
    remaining_stock INT          NOT NULL DEFAULT 0,
    sort_order      INT          NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_lottery_prize_weight CHECK (weight > 0),
    CONSTRAINT chk_lottery_prize_stock  CHECK (total_stock >= 0 AND remaining_stock >= 0)
);

CREATE INDEX idx_lottery_prizes_event ON lottery_prizes (event_id, sort_order);

CREATE TABLE lottery_records (
    id              BIGSERIAL    PRIMARY KEY,
    event_id        BIGINT       NOT NULL REFERENCES lottery_events(id),
    user_id         UUID         NOT NULL REFERENCES users(id),
    prize_id        BIGINT       NOT NULL REFERENCES lottery_prizes(id),
    spend_amount    NUMERIC(10,4) NOT NULL,
    ip_address      VARCHAR(45),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_lottery_record_user_event UNIQUE (event_id, user_id)
);

CREATE INDEX idx_lottery_records_user_event ON lottery_records (user_id, event_id);
CREATE INDEX idx_lottery_records_event      ON lottery_records (event_id, created_at DESC);
