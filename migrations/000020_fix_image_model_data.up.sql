-- Set existing image model pricing to image billing type
UPDATE model_pricing SET billing_type = 'image'
WHERE model_name IN ('gemini-3-pro-image-text-to-image', 'gemini-3-pro-image-edit');

-- Set image model categories
UPDATE models SET category = 'text-to-image'
WHERE name = 'gemini-3-pro-image-text-to-image' AND (category IS NULL OR category = '' OR category = 'chat');

UPDATE models SET category = 'image-edit'
WHERE name = 'gemini-3-pro-image-edit' AND (category IS NULL OR category = '' OR category = 'chat');
