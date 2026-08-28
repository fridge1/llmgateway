-- 为 upstreams 与 tenant_model_upstreams 增加数组形式的 protocols 字段，
-- 让一个上游能同时声明多种协议（例如聚合站 base_url 同时支持 openai 与 anthropic）。
-- 旧的单值 protocol 列保留：新代码优先读 protocols 数组，空数组时回退到 protocol 单值。
-- 蓝绿兼容：新列带 DEFAULT '{}'，旧 backend 不读不写该列仍可正常运行。
ALTER TABLE upstreams ADD COLUMN IF NOT EXISTS protocols TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE tenant_model_upstreams ADD COLUMN IF NOT EXISTS protocols TEXT[] NOT NULL DEFAULT '{}';

-- 把现有 protocol 单值搬进 protocols 数组（仅当数组为空且单值非空时）
UPDATE upstreams SET protocols = ARRAY[protocol] WHERE protocol <> '' AND protocols = '{}';
UPDATE tenant_model_upstreams SET protocols = ARRAY[protocol] WHERE protocol <> '' AND protocols = '{}';

-- GIN 索引便于将来按协议查询
CREATE INDEX IF NOT EXISTS idx_upstreams_protocols ON upstreams USING GIN (protocols);
CREATE INDEX IF NOT EXISTS idx_tenant_model_upstreams_protocols ON tenant_model_upstreams USING GIN (protocols);
