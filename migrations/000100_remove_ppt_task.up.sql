-- 删除「体验 PPT 生成」成长任务及用户进度记录。
DELETE FROM user_task_progress WHERE task_code = 'try_ppt';
DELETE FROM task_definitions WHERE code = 'try_ppt';
