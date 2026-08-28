-- 修复订阅升级时 trial 用量被重复扣除的历史数据
-- 对每条 active + 有 'upgrade' 订单 + 有早于等于 started_at 的 usage 行的订阅，
-- 把 stranded 用量累加回 extra_quota_cny
WITH stranded AS (
    SELECT su.subscription_id,
           SUM(su.amount_used) AS stranded_cny
    FROM subscription_usage su
    JOIN user_subscriptions us ON us.id = su.subscription_id
    WHERE us.status = 'active'
      AND su.period <= us.started_at::date
      AND EXISTS (
          SELECT 1 FROM subscription_orders so
          WHERE so.user_id = us.user_id
            AND so.plan_id = us.plan_id
            AND so.type = 'upgrade'
            AND so.status = 'paid'
      )
    GROUP BY su.subscription_id
    HAVING SUM(su.amount_used) > 0
)
UPDATE user_subscriptions us
SET extra_quota_cny = us.extra_quota_cny + s.stranded_cny,
    updated_at = NOW()
FROM stranded s
WHERE us.id = s.subscription_id;
