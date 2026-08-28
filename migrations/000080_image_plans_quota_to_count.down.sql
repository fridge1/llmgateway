-- 反向：图片套餐 quota_amount_cny 回到原始金额（元）
UPDATE subscription_plans SET quota_amount_cny = 32  WHERE name = 'image-basic';
UPDATE subscription_plans SET quota_amount_cny = 80  WHERE name = 'image-pro';
UPDATE subscription_plans SET quota_amount_cny = 160 WHERE name = 'image-max';

-- 历史用量从张数还原为元
UPDATE subscription_usage su
SET amount_used = amount_used * 0.30
FROM user_subscriptions us
JOIN subscription_plans sp ON sp.id = us.plan_id
WHERE su.subscription_id = us.id
  AND sp.name LIKE 'image-%';
