-- 为 api_keys 增加可空的 plan_id 外键，用于将 key 限制在某个套餐的模型范围内。
-- NULL 表示不限制（可访问全部模型），兼容所有存量 key。
ALTER TABLE api_keys ADD COLUMN plan_id INT REFERENCES subscription_plans(id);
