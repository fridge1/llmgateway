-- 移除美元汇率功能，所有定价统一使用人民币
DROP TABLE IF EXISTS fx_rates;

ALTER TABLE model_pricing DROP COLUMN IF EXISTS input_price_usd;
ALTER TABLE model_pricing DROP COLUMN IF EXISTS output_price_usd;
ALTER TABLE model_pricing DROP COLUMN IF EXISTS cached_input_price_usd;
ALTER TABLE model_pricing DROP COLUMN IF EXISTS cache_creation_price_usd;
ALTER TABLE model_pricing DROP COLUMN IF EXISTS cache_creation_1h_price_usd;
