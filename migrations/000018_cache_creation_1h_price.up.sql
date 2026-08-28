-- Add 1-hour cache creation price columns to model_pricing.
-- cache_creation_price (existing) = 5-minute ephemeral cache write price.
-- cache_creation_1h_price (new)   = 1-hour ephemeral cache write price.

ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS cache_creation_1h_price DECIMAL(10,6) NOT NULL DEFAULT 0;
ALTER TABLE model_pricing ADD COLUMN IF NOT EXISTS cache_creation_1h_price_usd DECIMAL(10,6) NOT NULL DEFAULT 0;

-- Default: 1h cache write price = input_price_usd × 2 (Anthropic ratio: Opus $10/$5)
UPDATE model_pricing SET cache_creation_1h_price_usd = input_price_usd * 2
WHERE cache_creation_1h_price_usd = 0 AND input_price_usd > 0;

-- Derive CNY from USD using current fx_rates
UPDATE model_pricing SET
    cache_creation_1h_price = cache_creation_1h_price_usd * (SELECT usd_cny FROM fx_rates WHERE id = 1)
WHERE cache_creation_1h_price_usd > 0;
