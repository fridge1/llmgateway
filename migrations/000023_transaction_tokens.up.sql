ALTER TABLE transactions
  ADD COLUMN prompt_tokens INTEGER,
  ADD COLUMN completion_tokens INTEGER,
  ADD COLUMN cache_read_tokens INTEGER,
  ADD COLUMN cache_creation_tokens INTEGER,
  ADD COLUMN cache_creation_5m_tokens INTEGER,
  ADD COLUMN cache_creation_1h_tokens INTEGER;
