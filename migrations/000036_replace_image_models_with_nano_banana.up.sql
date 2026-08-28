-- Replace old image models with new Gemini-style image generation model
-- This migration removes gemini-3-pro-image-text-to-image and gemini-3-pro-image-edit
-- and adds gemini-3-pro-image-preview using standard Gemini REST API format

BEGIN;

-- 1. Delete old model pricing configurations
DELETE FROM model_pricing
WHERE model_name IN ('gemini-3-pro-image-text-to-image', 'gemini-3-pro-image-edit');

-- 2. Delete old model upstreams
DELETE FROM upstreams
WHERE model_id IN (
    SELECT id FROM models
    WHERE name IN ('gemini-3-pro-image-text-to-image', 'gemini-3-pro-image-edit')
);

-- 3. Delete old models
DELETE FROM models
WHERE name IN ('gemini-3-pro-image-text-to-image', 'gemini-3-pro-image-edit');

-- 4. Create new image generation model
-- NOTE: User must update base_url and api_key with actual third-party API credentials
WITH m AS (
    INSERT INTO models (name, display_name, category)
    VALUES ('gemini-3-pro-image-preview', 'Gemini 3 Pro Image Preview', 'image-generation')
    RETURNING id
)
INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight)
SELECT
    id,
    'google',
    'REPLACE_WITH_THIRD_PARTY_BASE_URL',  -- User must replace this
    'REPLACE_WITH_API_KEY',                -- User must replace this
    'gemini-3-pro-image-preview',
    1
FROM m;

-- 5. Add pricing configuration (billing_type = 'image' for per-request billing)
-- NOTE: User must update input_price_usd with actual price per request (in CNY)
INSERT INTO model_pricing (
    model_name,
    input_price_usd,     -- Price per request (CNY)
    output_price_usd,    -- Not used for image billing, set same as input
    input_price,         -- Will be synced from input_price_usd
    output_price,        -- Will be synced from output_price_usd
    billing_type,
    is_active
)
VALUES (
    'gemini-3-pro-image-preview',
    0.10,        -- REPLACE: Set actual price per request in CNY
    0.10,        -- REPLACE: Set actual price per request in CNY
    0.10,
    0.10,
    'image',     -- Per-request billing
    true
);

COMMIT;
