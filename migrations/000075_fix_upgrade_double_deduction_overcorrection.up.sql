-- 修正 000074 的多补：000074 把 subscription_usage 当日整行 amount_used 当作 trial stranded 累加到了
-- extra_quota_cny，但同一行实际可能混了 "升级前 trial 消费 + 升级当天 Pro 升级后消费 + 之后所有 Pro 消费"
-- （因为旧 in-place 升级保留同一 subscription_id，currentPeriod 又只用 started_at::date，导致整个订阅期的
-- Pro 消费都写到同一行 (sub_id, model, started_at_date) 上）。
--
-- 这里用 transactions.created_at 精确算出"升级前消费" = 真实 stranded，
-- 加上 renew 订单累加的额度，得到 extra_quota_cny 应有的目标值，直接覆盖。

WITH upgrades AS (
    -- 每条 active 订阅最早的 upgrade 时间（in-place 升级后 sub_id 没变）
    SELECT us.id AS subscription_id,
           us.user_id,
           us.plan_id,
           MIN(so.created_at) AS upgrade_at
    FROM user_subscriptions us
    JOIN subscription_orders so
      ON so.user_id = us.user_id
     AND so.plan_id = us.plan_id
     AND so.type = 'upgrade'
     AND so.status = 'paid'
    WHERE us.status = 'active'
    GROUP BY us.id, us.user_id, us.plan_id
),
true_stranded AS (
    -- 升级时刻之前的真实订阅消费（按交易精确时间切）
    SELECT u.subscription_id,
           COALESCE(SUM(t.amount), 0) AS stranded_cny
    FROM upgrades u
    LEFT JOIN transactions t
      ON t.user_id = u.user_id
     AND t.subscription_id = u.subscription_id
     AND t.type = 'subscription_usage'
     AND t.created_at < u.upgrade_at
    GROUP BY u.subscription_id
),
renew_additions AS (
    -- renew 订单按当前 plan.quota_amount_cny 累加进 extra（与 AddSubscriptionQuota 行为一致）
    SELECT u.subscription_id,
           COALESCE(COUNT(so2.id), 0) * sp.quota_amount_cny AS renew_quota
    FROM upgrades u
    JOIN subscription_plans sp ON sp.id = u.plan_id
    LEFT JOIN subscription_orders so2
      ON so2.user_id = u.user_id
     AND so2.plan_id = u.plan_id
     AND so2.type = 'renew'
     AND so2.status = 'paid'
    GROUP BY u.subscription_id, sp.quota_amount_cny
),
target AS (
    SELECT s.subscription_id,
           ROUND((s.stranded_cny + r.renew_quota)::numeric, 4) AS target_extra
    FROM true_stranded s
    JOIN renew_additions r ON r.subscription_id = s.subscription_id
)
UPDATE user_subscriptions us
SET extra_quota_cny = GREATEST(t.target_extra, 0),
    updated_at = NOW()
FROM target t
WHERE us.id = t.subscription_id
  AND us.extra_quota_cny > t.target_extra;
