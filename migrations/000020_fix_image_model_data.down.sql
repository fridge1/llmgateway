-- Revert image models to token billing type
UPDATE model_pricing SET billing_type = 'token'
WHERE model_name IN ('gemini-3-pro-image-text-to-image', 'gemini-3-pro-image-edit');

UPDATE models SET category = ''
WHERE name IN ('gemini-3-pro-image-text-to-image', 'gemini-3-pro-image-edit');
