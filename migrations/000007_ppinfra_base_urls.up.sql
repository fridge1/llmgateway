-- Update all upstream base_urls to use ppinfra (派欧云) endpoints
-- and add missing model pa/claude-sonnet-4-5-20250929-1m

BEGIN;

-- ============================================================
-- Update base_urls: all models go through ppinfra OpenAI-compatible endpoint
-- Gateway appends r.URL.Path (/v1/chat/completions) to base_url,
-- so base_url = https://api.ppinfra.com/openai produces the correct
-- upstream URL: https://api.ppinfra.com/openai/v1/chat/completions
-- ============================================================

-- OpenAI 系列
UPDATE upstreams SET base_url = 'https://api.ppinfra.com/openai'
WHERE provider = 'openai';

-- Anthropic 系列 (via ppinfra OpenAI-compatible endpoint)
UPDATE upstreams SET base_url = 'https://api.ppinfra.com/openai'
WHERE provider = 'anthropic';

-- Google 系列 (via ppinfra OpenAI-compatible endpoint)
UPDATE upstreams SET base_url = 'https://api.ppinfra.com/openai'
WHERE provider = 'google';

-- Grok/xAI 系列
UPDATE upstreams SET base_url = 'https://api.ppinfra.com/openai'
WHERE provider = 'xai';

-- 豆包/Volcengine 系列
UPDATE upstreams SET base_url = 'https://api.ppinfra.com/openai'
WHERE provider = 'volcengine';

-- ============================================================
-- Add missing model: Claude Sonnet 4.5 1M input version
-- ============================================================
WITH m AS (
    INSERT INTO models (name) VALUES ('pa/claude-sonnet-4-5-20250929-1m')
    ON CONFLICT (name) DO NOTHING
    RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight)
SELECT id, 'anthropic', 'https://api.ppinfra.com/openai',
       (SELECT api_key FROM upstreams WHERE model_id = (SELECT id FROM models WHERE name = 'pa/claude-sonnet-4-5-20250929') LIMIT 1),
       'claude-sonnet-4-5-20250929', 1
FROM m;

COMMIT;
