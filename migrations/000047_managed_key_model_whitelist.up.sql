ALTER TABLE managed_api_keys
ADD COLUMN allowed_models TEXT[] DEFAULT NULL;
