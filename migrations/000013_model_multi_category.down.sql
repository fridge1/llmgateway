-- Revert all categories back to 'chat' and restore VARCHAR(50)
BEGIN;

UPDATE models SET category = 'chat';
ALTER TABLE models ALTER COLUMN category TYPE VARCHAR(50);

COMMIT;
