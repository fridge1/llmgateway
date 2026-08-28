-- Rollback IP Blocking table
DROP INDEX IF EXISTS idx_blocked_ips_blocked_at;
DROP INDEX IF EXISTS idx_blocked_ips_expires;
DROP TABLE IF EXISTS blocked_ips;
