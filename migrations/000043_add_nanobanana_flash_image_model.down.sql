BEGIN;

DELETE FROM model_pricing WHERE model_name = 'gemini-3.1-flash-image-preview';

DELETE FROM upstreams WHERE model_id IN (
    SELECT id FROM models WHERE name = 'gemini-3.1-flash-image-preview'
);

DELETE FROM models WHERE name = 'gemini-3.1-flash-image-preview';

COMMIT;
