-- Add pricing_tiers JSONB column for tiered pricing (e.g. GLM-5.1).
-- Format: [{"min_tokens":1,"max_tokens":32768,"input_price":6,"output_price":24,"cached_input_price":1.3}, ...]
-- Prices in tiers are CNY per 1M tokens. When non-null, calculateCost uses tiers instead of flat prices.
ALTER TABLE model_pricing ADD COLUMN pricing_tiers JSONB;
