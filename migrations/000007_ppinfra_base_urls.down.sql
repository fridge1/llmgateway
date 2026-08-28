-- Rollback: restore original base_urls and remove 1M model

BEGIN;

-- Restore original base_urls
UPDATE upstreams SET base_url = 'https://api.openai.com/v1'
WHERE provider = 'openai';

UPDATE upstreams SET base_url = 'https://api.anthropic.com/v1'
WHERE provider = 'anthropic';

UPDATE upstreams SET base_url = 'https://generativelanguage.googleapis.com/v1beta'
WHERE provider = 'google';

UPDATE upstreams SET base_url = 'https://api.x.ai/v1'
WHERE provider = 'xai';

UPDATE upstreams SET base_url = 'https://ark.cn-beijing.volces.com/api/v3'
WHERE provider = 'volcengine';

-- Remove 1M model
DELETE FROM models WHERE name = 'pa/claude-sonnet-4-5-20250929-1m';

COMMIT;
