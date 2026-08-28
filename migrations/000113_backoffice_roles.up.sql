-- Back-office RBAC: extend role beyond user/admin with support/finance/ops.
-- No CHECK constraint is added (keeps blue-green compatibility and matches the
-- existing schema which relies on application-level validation).
-- This migration only documents the new values; no schema change is needed
-- because users.role is VARCHAR(20) without a constraint.
SELECT 1;
