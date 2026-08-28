-- backfill_transaction_model.sql
-- 用于回填历史 transactions 中 model 名与 model_pricing 不匹配的记录。
-- 请先执行 Step 1 审查映射关系，确认无歧义后再执行 Step 2。

-- Step 1: 查看需要回填的数据（只读，不修改）
SELECT
    t.model        AS old_name,
    p.model_name   AS new_name,
    COUNT(*)       AS row_count
FROM transactions t
JOIN model_pricing p
  ON split_part(p.model_name, '/', 2) = split_part(t.model, '/', 2)
WHERE t.type = 'consumption'
  AND t.model IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM model_pricing WHERE model_name = t.model)
GROUP BY t.model, p.model_name
ORDER BY row_count DESC;

-- Step 2: 确认上面的结果无歧义（每个 old_name 只对应一个 new_name）后，
-- 取消下面的注释并执行回填。
--
-- UPDATE transactions t
-- SET model = p.model_name
-- FROM model_pricing p
-- WHERE t.type = 'consumption'
--   AND t.model IS NOT NULL
--   AND NOT EXISTS (SELECT 1 FROM model_pricing WHERE model_name = t.model)
--   AND split_part(p.model_name, '/', 2) = split_part(t.model, '/', 2);
