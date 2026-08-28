-- 回滚到加量前的额度
UPDATE subscription_plans SET quota_amount_cny = 499, updated_at = NOW() WHERE name = 'plus';
UPDATE subscription_plans SET quota_amount_cny = 998, updated_at = NOW() WHERE name = 'premium';
