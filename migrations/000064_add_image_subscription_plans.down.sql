DELETE FROM subscription_plan_models WHERE plan_id IN (SELECT id FROM subscription_plans WHERE name IN ('image-basic', 'image-pro', 'image-max'));
DELETE FROM subscription_plans WHERE name IN ('image-basic', 'image-pro', 'image-max');

-- 恢复 gpt-image-2 的 4K 差异定价
UPDATE model_pricing SET output_price = 0.16, updated_at = NOW()
WHERE model_name = 'gpt-image-2';
