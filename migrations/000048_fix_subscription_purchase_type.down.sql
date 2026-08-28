-- Revert sub_purchase back to consumption.
UPDATE transactions
SET type = 'consumption'
WHERE type = 'sub_purchase';
