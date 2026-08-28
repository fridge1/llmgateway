-- Rollback: Remove new model and restore old models (structure only, no data)

BEGIN;

-- Delete new model
DELETE FROM model_pricing WHERE model_name = 'gemini-3-pro-image-preview';

DELETE FROM upstreams WHERE model_id IN (
    SELECT id FROM models WHERE name = 'gemini-3-pro-image-preview'
);

DELETE FROM models WHERE name = 'gemini-3-pro-image-preview';

-- Note: This rollback does NOT restore the old models or their data
-- If you need to restore old models, you must manually recreate them

COMMIT;
