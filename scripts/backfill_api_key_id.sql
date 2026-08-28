-- 历史数据修复脚本：回填 transactions 表中缺失的 api_key_id
--
-- 背景：
-- - 总计 69,908 条记录的 api_key_id 为 NULL（11.9%）
-- - 其中 8,107 条来自租户成员（11.6%）
-- - 其中 61,804 条来自普通用户（88.4%）
--
-- 策略：
-- 1. 对于只有 1 个活跃 key 的用户：直接回填该 key
-- 2. 对于有多个 key 的用户：无法准确推断，保持 NULL
--
-- 风险：
-- - 如果用户在历史时期删除了某个 key，可能导致误判
-- - 建议在测试环境先验证
--
-- 使用方法：
-- psql "postgres://gateway:PASSWORD@HOST:5432/gateway" -f backfill_api_key_id.sql

BEGIN;

-- 创建临时表存储可回填的记录
CREATE TEMP TABLE backfill_candidates AS
WITH user_key_counts AS (
    -- 统计每个用户的活跃 key 数量
    SELECT user_id, COUNT(*) as key_count
    FROM api_keys
    WHERE deleted_at IS NULL
    GROUP BY user_id
    HAVING COUNT(*) = 1
),
single_key_users AS (
    -- 获取只有 1 个活跃 key 的用户及其 key
    SELECT ak.user_id, ak.id as api_key_id
    FROM api_keys ak
    JOIN user_key_counts ukc ON ak.user_id = ukc.user_id
    WHERE ak.deleted_at IS NULL
)
SELECT
    t.id as transaction_id,
    t.user_id,
    sku.api_key_id
FROM transactions t
JOIN single_key_users sku ON t.user_id = sku.user_id
WHERE t.type IN ('consumption', 'subscription_usage')
  AND t.api_key_id IS NULL
  AND t.user_id NOT IN (
      -- 排除租户成员（他们可能有多个租户的不同 key）
      SELECT user_id FROM tenant_members
  );

-- 显示回填统计
SELECT
    COUNT(*) as total_backfillable,
    COUNT(DISTINCT user_id) as affected_users,
    ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM transactions WHERE api_key_id IS NULL), 2) as percentage_of_nulls
FROM backfill_candidates;

-- 确认是否继续（手动执行时）
-- 如果确认，执行下面的 UPDATE；否则执行 ROLLBACK;

-- 执行回填
UPDATE transactions t
SET api_key_id = bc.api_key_id
FROM backfill_candidates bc
WHERE t.id = bc.transaction_id;

-- 显示回填结果
SELECT
    '回填完成' as status,
    COUNT(*) as updated_count
FROM backfill_candidates;

-- 显示剩余的 NULL 记录统计
SELECT
    '剩余 NULL 记录' as status,
    COUNT(*) as remaining_nulls,
    COUNT(DISTINCT user_id) as affected_users
FROM transactions
WHERE type IN ('consumption', 'subscription_usage')
  AND api_key_id IS NULL;

-- 取消注释下面一行以提交更改
-- COMMIT;

-- 默认回滚（安全起见）
ROLLBACK;

-- 使用说明：
-- 1. 先在测试环境运行，检查 backfill_candidates 的数量是否合理
-- 2. 如果确认无误，将最后的 ROLLBACK 改为 COMMIT
-- 3. 在生产环境运行前，建议先备份 transactions 表
