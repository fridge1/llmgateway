-- 图片套餐配额从「金额（元）」改为「张数」
-- 套餐 quota_amount_cny 字段语义切换：image-* 套餐下当作"张数"读
UPDATE subscription_plans SET quota_amount_cny = 400  WHERE name = 'image-basic';
UPDATE subscription_plans SET quota_amount_cny = 1000 WHERE name = 'image-pro';
UPDATE subscription_plans SET quota_amount_cny = 2000 WHERE name = 'image-max';

-- 历史 subscription_usage.amount_used 是元（按 0.30 元/张扣费），换算回张数
-- 仅处理图片订阅。当前数据库中图片订阅只用过 gpt-image-2（单价 0.30）
UPDATE subscription_usage su
SET amount_used = ROUND(amount_used / 0.30)
FROM user_subscriptions us
JOIN subscription_plans sp ON sp.id = us.plan_id
WHERE su.subscription_id = us.id
  AND sp.name LIKE 'image-%';
