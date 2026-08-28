-- Backfill tenant_balances.total_consumed from tenant_transactions history.
-- Previously, sub-user consumption was not syncing total_consumed.
UPDATE tenant_balances tb
SET total_consumed = COALESCE(sub.total, 0),
    updated_at = NOW()
FROM (
    SELECT tenant_id, SUM(amount) AS total
    FROM tenant_transactions
    WHERE type = 'consumption'
    GROUP BY tenant_id
) sub
WHERE tb.tenant_id = sub.tenant_id;
