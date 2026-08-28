-- Add cache_creation_price columns to model_pricing.
-- cache_creation_price: CNY per 1M tokens for cache-write (prompt cache miss → create).
-- cache_creation_price_usd: USD per 1M tokens (source of truth; CNY derived via fx_rates).

BEGIN;

ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS cache_creation_price DECIMAL(10,6) NOT NULL DEFAULT 0;
ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS cache_creation_price_usd DECIMAL(10,6) NOT NULL DEFAULT 0;

-- Populate cache_creation_price_usd based on official ratios:
-- Anthropic: cache_creation = input × 1.25
-- OpenAI:    cache_creation = input × 1.0  (no separate creation premium)
-- Google:    cache_creation = input × 1.0
-- xAI:      cache_creation = input × 1.0
-- Doubao:   cache_creation = input × 1.0

-- Anthropic models (1.25× input)
UPDATE model_pricing SET cache_creation_price_usd = input_price_usd * 1.25
WHERE model_name LIKE 'pa/claude-%' OR model_name LIKE 'pa/cd-%';

-- All other models (same as input_price — no separate cache creation premium)
UPDATE model_pricing SET cache_creation_price_usd = input_price_usd
WHERE cache_creation_price_usd = 0 AND input_price_usd > 0;

-- Derive CNY from USD using current fx rate
UPDATE model_pricing SET
    cache_creation_price = cache_creation_price_usd * (SELECT usd_cny FROM fx_rates WHERE id = 1)
WHERE cache_creation_price_usd > 0;

COMMIT;
