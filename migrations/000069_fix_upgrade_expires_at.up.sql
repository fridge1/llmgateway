-- 修复 2026-04-22 升级 bug 导致的到期日期错误
-- 受影响：两个在 94cefd5 修复前从 trial 升级到 pro 的用户
-- 问题：旧版 UpgradeSubscription 只更新了 plan_id，未更新 expires_at
-- 修复：将 expires_at 更正为 started_at + duration_days，并恢复 active 状态

UPDATE user_subscriptions us
SET expires_at = us.started_at + (sp.duration_days || ' days')::interval,
    status = 'active',
    updated_at = NOW()
FROM subscription_plans sp
WHERE sp.id = us.plan_id
  AND us.expires_at < us.started_at + (sp.duration_days || ' days')::interval - INTERVAL '1 minute'
  AND us.started_at + (sp.duration_days || ' days')::interval > NOW();
