-- Revert subscription plan display names and descriptions to original
UPDATE subscription_plans SET
    display_name = 'Trial',
    description = '体验版套餐，7 天有效期。'
WHERE name = 'trial';

UPDATE subscription_plans SET
    display_name = 'Pro',
    description = '专业版套餐，适合个人开发者。'
WHERE name = 'pro';

UPDATE subscription_plans SET
    display_name = 'Plus',
    description = '增强版套餐，适合重度用户。'
WHERE name = 'plus';

UPDATE subscription_plans SET
    display_name = 'Premium',
    description = '高级版套餐，适合团队使用。'
WHERE name = 'premium';

UPDATE subscription_plans SET
    display_name = 'Max',
    description = '旗舰版套餐，不限量使用。'
WHERE name = 'max';
