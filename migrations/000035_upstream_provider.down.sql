-- 回滚上游供应商追踪字段
ALTER TABLE upstreams
DROP COLUMN IF EXISTS upstream_name,
DROP COLUMN IF EXISTS upstream_provider;
