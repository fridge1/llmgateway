-- 添加上游供应商追踪字段
ALTER TABLE upstreams
ADD COLUMN IF NOT EXISTS upstream_provider VARCHAR(100) DEFAULT '',
ADD COLUMN IF NOT EXISTS upstream_name VARCHAR(255) DEFAULT '';

-- 为现有数据填充默认值（基于base_url推断）
UPDATE upstreams
SET upstream_provider = 'ppinfra',
    upstream_name = 'PPInfra平台'
WHERE base_url LIKE '%ppinfra%' AND upstream_provider = '';
