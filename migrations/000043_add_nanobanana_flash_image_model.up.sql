-- Add gemini-3.1-flash-image-preview for nanobanana image editing API

BEGIN;

WITH m AS (
    INSERT INTO models (name, display_name, category)
    VALUES ('gemini-3.1-flash-image-preview', 'Gemini 3.1 Flash Image Preview', 'image-generation')
    RETURNING id
)
INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight)
SELECT
    id,
    'google',
    'REPLACE_WITH_THIRD_PARTY_BASE_URL',
    'REPLACE_WITH_API_KEY',
    'gemini-3.1-flash-image-preview',
    1
FROM m;

INSERT INTO model_pricing (
    model_name,
    input_price,
    output_price,
    billing_type,
    is_active
)
VALUES (
    'gemini-3.1-flash-image-preview',
    0.10,
    0.10,
    'image',
    true
);

COMMIT;
