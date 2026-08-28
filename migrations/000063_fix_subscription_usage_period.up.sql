-- Fix subscription_usage period: merge calendar-month periods into subscription started_at::date.
-- This is a lossy operation (multiple month rows merged into one), so the down migration is a no-op.

BEGIN;

-- Step 1: Create a temp table with the merged usage per (subscription_id, model_name) using started_at::date as period.
CREATE TEMP TABLE _merged_usage AS
SELECT su.subscription_id,
       su.user_id,
       su.model_name,
       us.started_at::date AS period,
       SUM(su.input_tokens_used) AS input_tokens_used,
       SUM(su.output_tokens_used) AS output_tokens_used,
       SUM(su.cache_read_tokens_used) AS cache_read_tokens_used,
       SUM(su.cache_creation_tokens_used) AS cache_creation_tokens_used,
       SUM(su.amount_used) AS amount_used,
       SUM(su.request_count) AS request_count
FROM subscription_usage su
JOIN user_subscriptions us ON us.id = su.subscription_id
WHERE su.period != us.started_at::date
GROUP BY su.subscription_id, su.user_id, su.model_name, us.started_at::date;

-- Step 2: Delete the old rows whose period doesn't match started_at::date.
DELETE FROM subscription_usage su
USING user_subscriptions us
WHERE su.subscription_id = us.id AND su.period != us.started_at::date;

-- Step 3: Upsert the merged data. If a row with the correct period already exists, add to it.
INSERT INTO subscription_usage (subscription_id, user_id, model_name, period,
    input_tokens_used, output_tokens_used, cache_read_tokens_used,
    cache_creation_tokens_used, amount_used, request_count, updated_at)
SELECT subscription_id, user_id, model_name, period,
       input_tokens_used, output_tokens_used, cache_read_tokens_used,
       cache_creation_tokens_used, amount_used, request_count, NOW()
FROM _merged_usage
ON CONFLICT (subscription_id, model_name, period)
DO UPDATE SET
    input_tokens_used = subscription_usage.input_tokens_used + EXCLUDED.input_tokens_used,
    output_tokens_used = subscription_usage.output_tokens_used + EXCLUDED.output_tokens_used,
    cache_read_tokens_used = subscription_usage.cache_read_tokens_used + EXCLUDED.cache_read_tokens_used,
    cache_creation_tokens_used = subscription_usage.cache_creation_tokens_used + EXCLUDED.cache_creation_tokens_used,
    amount_used = subscription_usage.amount_used + EXCLUDED.amount_used,
    request_count = subscription_usage.request_count + EXCLUDED.request_count,
    updated_at = NOW();

DROP TABLE _merged_usage;

COMMIT;
