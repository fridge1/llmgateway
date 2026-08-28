CREATE TABLE lottery_events (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT DEFAULT '',
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  min_recharge_cny DECIMAL(10,2) NOT NULL DEFAULT 1.00,
  start_time TIMESTAMP,
  end_time TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE lottery_prizes (
  id SERIAL PRIMARY KEY,
  event_id INT NOT NULL REFERENCES lottery_events(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  description TEXT DEFAULT '',
  weight INT NOT NULL DEFAULT 100,
  total_stock INT NOT NULL DEFAULT 0,
  remaining_stock INT NOT NULL DEFAULT 0,
  prize_type VARCHAR(20) NOT NULL DEFAULT 'none',
  prize_value DECIMAL(10,2) NOT NULL DEFAULT 0,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE lottery_records (
  id BIGSERIAL PRIMARY KEY,
  event_id INT NOT NULL REFERENCES lottery_events(id),
  user_id VARCHAR(255) NOT NULL,
  prize_id INT REFERENCES lottery_prizes(id),
  order_no VARCHAR(255) NOT NULL,
  recharge_amount DECIMAL(10,2) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX lottery_records_order_no_idx ON lottery_records(order_no);
