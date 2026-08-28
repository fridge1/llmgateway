-- 抽奖活动新增"开奖所需最低笔数"门槛：未达标禁止开奖
ALTER TABLE lottery_events
  ADD COLUMN IF NOT EXISTS min_order_count_to_draw INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN lottery_events.min_order_count_to_draw IS '开奖所需最低达标订单笔数；0 表示不限制';
