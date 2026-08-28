-- 抽奖活动幂等开奖：记录开奖时间与操作人，防止重复开奖
ALTER TABLE lottery_events
  ADD COLUMN IF NOT EXISTS drawn_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS drawn_by UUID REFERENCES users(id);

COMMENT ON COLUMN lottery_events.drawn_at IS '开奖完成时间；NULL 表示未开奖';
COMMENT ON COLUMN lottery_events.drawn_by IS '执行开奖的管理员 ID';
