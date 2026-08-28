-- Reclassify historical subscription purchase transactions from 'consumption' to 'sub_purchase'.
-- These are identified by their description containing '订阅套餐' or '升级套餐'.
UPDATE transactions
SET type = 'sub_purchase'
WHERE type = 'consumption'
  AND (description LIKE '订阅套餐%' OR description LIKE '升级套餐%');
