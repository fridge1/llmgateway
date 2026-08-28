-- Rollback: restore original model names without pa/ prefix

BEGIN;

UPDATE upstreams SET model_override = regexp_replace(model_override, '^pa/', '');

COMMIT;
