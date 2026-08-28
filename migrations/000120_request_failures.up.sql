CREATE TABLE request_failures (
  id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID         NOT NULL REFERENCES users(id),
  model       VARCHAR(100) NOT NULL,
  http_status INT          NOT NULL DEFAULT 503,
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_request_failures_user_model_created
  ON request_failures (user_id, model, created_at DESC);

CREATE INDEX idx_request_failures_created_at
  ON request_failures (created_at DESC);
