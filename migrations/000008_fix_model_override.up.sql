-- Fix model_override: ppinfra expects pa/ prefixed model names
-- The gateway replaces model name with model_override before sending upstream.
-- Since ppinfra uses pa/ model names, model_override should match the models.name.

BEGIN;

UPDATE upstreams u
SET model_override = m.name
FROM models m
WHERE u.model_id = m.id;

COMMIT;
