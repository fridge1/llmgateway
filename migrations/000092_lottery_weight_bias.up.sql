-- 为 lottery_events 增加"低充值倾斜系数 k"，用于 post（后台批量开奖）模式按充值额反向加权选中奖者。
-- weight_bias_k = 0 表示沿用旧的均匀随机（向后兼容存量活动）；> 0 启用反向加权：
-- 权重 w_i = 1 / (spend_i + k)，k 越大越接近均匀，越小越偏向低充值用户。
ALTER TABLE lottery_events
  ADD COLUMN weight_bias_k NUMERIC(12,4) NOT NULL DEFAULT 0;
