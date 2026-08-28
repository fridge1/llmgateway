-- 加量不加价：提高 Claude plus / premium 套餐额度，修正折扣倒挂
-- plus:    299 元 → 544 额度（约 5.5 折）
-- premium: 599 元 → 1152 额度（约 5.2 折）
UPDATE subscription_plans SET quota_amount_cny = 544, updated_at = NOW() WHERE name = 'plus';
UPDATE subscription_plans SET quota_amount_cny = 1152, updated_at = NOW() WHERE name = 'premium';
