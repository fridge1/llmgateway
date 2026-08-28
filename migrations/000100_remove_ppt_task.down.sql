-- 回滚：重新插入「体验 PPT 生成」任务定义（进度记录不可恢复）。
INSERT INTO task_definitions (code, title, description, reward_cny, sort_order)
VALUES ('try_ppt', '体验 PPT 生成', '使用 PPT 生成功能创建一份演示', 1.0, 5)
ON CONFLICT (code) DO NOTHING;
