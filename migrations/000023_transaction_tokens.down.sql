ALTER TABLE transactions
  DROP COLUMN IF EXISTS prompt_tokens,
  DROP COLUMN IF EXISTS completion_tokens,
  DROP COLUMN IF EXISTS cache_read_tokens,
  DROP COLUMN IF EXISTS cache_creation_tokens,
  DROP COLUMN IF EXISTS cache_creation_5m_tokens,
  DROP COLUMN IF EXISTS cache_creation_1h_tokens;
