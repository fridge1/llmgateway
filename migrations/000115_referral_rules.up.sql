-- Referral rule engine: DB-configurable reward amounts replacing config-only
-- values. Single active rule at a time (latest effective wins).
CREATE TABLE IF NOT EXISTS referral_rules (
    id SERIAL PRIMARY KEY,
    inviter_bonus_cny DECIMAL(12,2) NOT NULL DEFAULT 0,
    invitee_bonus_cny DECIMAL(12,2) NOT NULL DEFAULT 0,
    min_first_recharge_cny DECIMAL(12,2) NOT NULL DEFAULT 0,  -- 0 = any first recharge
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_referral_rules_effective ON referral_rules (effective_from DESC);
