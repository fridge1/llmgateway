-- Update subscription plan display names and descriptions to Claude-focused branding
UPDATE subscription_plans SET
    display_name = 'Claude 体验版',
    description = '体验周卡，适合新用户试用'
WHERE name = 'trial';

UPDATE subscription_plans SET
    display_name = 'Claude 开发者版',
    description = '入门订阅，适合轻度使用'
WHERE name = 'pro';

UPDATE subscription_plans SET
    display_name = 'Claude 专业版',
    description = '专业订阅，适合日常开发'
WHERE name = 'plus';

UPDATE subscription_plans SET
    display_name = 'Claude 团队版',
    description = '高级订阅，适合中高强度使用'
WHERE name = 'premium';

UPDATE subscription_plans SET
    display_name = 'Claude 无限版',
    description = '旗舰订阅，重度使用'
WHERE name = 'max';
