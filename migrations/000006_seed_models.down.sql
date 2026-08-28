-- Rollback: remove all seeded models (cascades to upstreams)
BEGIN;

DELETE FROM models WHERE name LIKE 'pa/%';

COMMIT;
