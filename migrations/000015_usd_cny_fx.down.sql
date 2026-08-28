ALTER TABLE model_pricing DROP COLUMN IF EXISTS input_price_usd;
ALTER TABLE model_pricing DROP COLUMN IF EXISTS output_price_usd;
ALTER TABLE model_pricing DROP COLUMN IF EXISTS cached_input_price_usd;
DROP TABLE IF EXISTS fx_rates;
