-- Add cached_input_price column to model_pricing
-- This stores the per-1M-token price for cached (prompt cache hit) input tokens.

BEGIN;

ALTER TABLE model_pricing ADD COLUMN cached_input_price DECIMAL(10,6) NOT NULL DEFAULT 0;

-- Populate cached_input_price for all existing models (official cache price × 0.8)
-- OpenAI GPT-5.x: cached = input × 10% (90% off)
-- OpenAI GPT-4.1/4o/o-series: cached = input × 25%~50% (varies)
-- Anthropic: cached = input × 10%
-- Google Gemini: cached = input × 25%
-- xAI Grok: cached = input × 25%
-- Doubao: cached = input × 10%

-- OpenAI GPT-5.4 系列 (cache = 10% of input)
UPDATE model_pricing SET cached_input_price = 2.400000 WHERE model_name = 'pa/gpt-5.4-pro';
UPDATE model_pricing SET cached_input_price = 0.200000 WHERE model_name = 'pa/gpt-5.4';
-- OpenAI GPT-5.3
UPDATE model_pricing SET cached_input_price = 0.200000 WHERE model_name = 'pa/gpt-5.3-chat-latest';
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5.3-codex';
-- OpenAI GPT-5.2
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5.2-codex';
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5.2';
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5.2-chat-latest';
UPDATE model_pricing SET cached_input_price = 2.400000 WHERE model_name = 'pa/gpt-5.2-pro';
-- OpenAI GPT-5.1
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5.1-codex-max';
UPDATE model_pricing SET cached_input_price = 0.020000 WHERE model_name = 'pa/gpt-5.1-codex-mini';
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5.1';
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5.1-chat-latest';
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5.1-codex';
-- OpenAI GPT-5
UPDATE model_pricing SET cached_input_price = 2.400000 WHERE model_name = 'pa/gpt-5-pro';
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5-codex';
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5';
UPDATE model_pricing SET cached_input_price = 0.020000 WHERE model_name = 'pa/gpt-5-mini';
UPDATE model_pricing SET cached_input_price = 0.004000 WHERE model_name = 'pa/gpt-5-nano';
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gpt-5-chat-latest';
-- OpenAI GPT-4.1 (cache = 25% of input)
UPDATE model_pricing SET cached_input_price = 0.400000 WHERE model_name = 'pa/gt-4.1';
UPDATE model_pricing SET cached_input_price = 0.020000 WHERE model_name = 'pa/gt-4.1-n';
UPDATE model_pricing SET cached_input_price = 0.080000 WHERE model_name = 'pa/gt-4.1-m';
-- OpenAI GPT-4o (cache = 50% of input)
UPDATE model_pricing SET cached_input_price = 1.000000 WHERE model_name = 'pa/gt-4p';
UPDATE model_pricing SET cached_input_price = 0.060000 WHERE model_name = 'pa/gt-4p-m';
-- OpenAI o-series (cache = 50% of input)
UPDATE model_pricing SET cached_input_price = 6.000000 WHERE model_name = 'pa/p1';
UPDATE model_pricing SET cached_input_price = 1.200000 WHERE model_name = 'pa/p1-m';
UPDATE model_pricing SET cached_input_price = 0.440000 WHERE model_name = 'pa/p3-m';
UPDATE model_pricing SET cached_input_price = 0.800000 WHERE model_name = 'pa/p3';
UPDATE model_pricing SET cached_input_price = 0.800000 WHERE model_name = 'pa/o4-mini';
-- OpenAI Embedding (no cache)
UPDATE model_pricing SET cached_input_price = 0.000000 WHERE model_name = 'pa/text-embedding-3-large';

-- Anthropic (cache read = 10% of input)
UPDATE model_pricing SET cached_input_price = 0.240000 WHERE model_name = 'pa/claude-sonnet-4-6';
UPDATE model_pricing SET cached_input_price = 0.400000 WHERE model_name = 'pa/claude-opus-4-6';
UPDATE model_pricing SET cached_input_price = 0.400000 WHERE model_name = 'pa/claude-opus-4-5-20251101';
UPDATE model_pricing SET cached_input_price = 0.240000 WHERE model_name = 'pa/claude-sonnet-4-5-20250929';
UPDATE model_pricing SET cached_input_price = 0.240000 WHERE model_name = 'pa/claude-sonnet-4-5-20250929-1m';
UPDATE model_pricing SET cached_input_price = 0.080000 WHERE model_name = 'pa/claude-haiku-4-5-20251001';
UPDATE model_pricing SET cached_input_price = 1.200000 WHERE model_name = 'pa/claude-opus-4-1-20250805';
UPDATE model_pricing SET cached_input_price = 0.240000 WHERE model_name = 'pa/cd-st-4-20250514';
UPDATE model_pricing SET cached_input_price = 1.200000 WHERE model_name = 'pa/cd-op-4-20250514';
UPDATE model_pricing SET cached_input_price = 0.240000 WHERE model_name = 'pa/cd-3-7-st-20250219';
UPDATE model_pricing SET cached_input_price = 0.240000 WHERE model_name = 'pa/cd-3-5-st-20241022';
UPDATE model_pricing SET cached_input_price = 0.064000 WHERE model_name = 'pa/cd-3-5-hk-20241022';
UPDATE model_pricing SET cached_input_price = 0.020000 WHERE model_name = 'pa/cd-3-hk-20240307';

-- Google Gemini (cache read = 25% of input)
UPDATE model_pricing SET cached_input_price = 0.060000 WHERE model_name = 'pa/gmn-2.5-fls';
UPDATE model_pricing SET cached_input_price = 0.250000 WHERE model_name = 'pa/gmn-2.5-pr';
UPDATE model_pricing SET cached_input_price = 0.020000 WHERE model_name = 'pa/gmn-2.5-fls-lt';
UPDATE model_pricing SET cached_input_price = 0.020000 WHERE model_name = 'pa/gmn-2.0-fls-20250609';
UPDATE model_pricing SET cached_input_price = 0.020000 WHERE model_name = 'pa/gmn-2.0-fls-lt';
UPDATE model_pricing SET cached_input_price = 0.060000 WHERE model_name = 'pa/gmn-2.5-fls-pw-05-20';
UPDATE model_pricing SET cached_input_price = 0.250000 WHERE model_name = 'pa/gmn-2.5-pr-pw-06-05';
UPDATE model_pricing SET cached_input_price = 0.020000 WHERE model_name = 'pa/gmn-2.5-fls-lt-pw-06-17';
UPDATE model_pricing SET cached_input_price = 0.020000 WHERE model_name = 'pa/gemini-2.5-flash-lite-preview';
UPDATE model_pricing SET cached_input_price = 0.400000 WHERE model_name = 'pa/gemini-3-pro-preview';
UPDATE model_pricing SET cached_input_price = 0.100000 WHERE model_name = 'pa/gemini-3-flash-preview';
UPDATE model_pricing SET cached_input_price = 0.400000 WHERE model_name = 'pa/gemini-3.1-pro-preview';
UPDATE model_pricing SET cached_input_price = 0.020000 WHERE model_name = 'pa/gemini-3.1-flash-lite-preview';

-- xAI Grok (cache = 25% of input)
UPDATE model_pricing SET cached_input_price = 0.040000 WHERE model_name = 'pa/grok-4-1-fast-non-reasoning';
UPDATE model_pricing SET cached_input_price = 0.040000 WHERE model_name = 'pa/grok-4-1-fast-reasoning';
UPDATE model_pricing SET cached_input_price = 0.600000 WHERE model_name = 'pa/grk-4';
UPDATE model_pricing SET cached_input_price = 0.040000 WHERE model_name = 'pa/grok-4-fast-reasoning';
UPDATE model_pricing SET cached_input_price = 0.040000 WHERE model_name = 'pa/grok-4-fast-non-reasoning';
UPDATE model_pricing SET cached_input_price = 0.040000 WHERE model_name = 'pa/grok-code-fast-1';
UPDATE model_pricing SET cached_input_price = 0.600000 WHERE model_name = 'pa/grk-3';
UPDATE model_pricing SET cached_input_price = 0.060000 WHERE model_name = 'pa/grok-3-mini';

-- Doubao/Volcengine (cache = 10% of input)
UPDATE model_pricing SET cached_input_price = 0.009000 WHERE model_name = 'pa/doubao-seed-1-8-251228';
UPDATE model_pricing SET cached_input_price = 0.009000 WHERE model_name = 'pa/doubao-seed-1.6';
UPDATE model_pricing SET cached_input_price = 0.009000 WHERE model_name = 'pa/doubao-seed-1.6-thinking';
UPDATE model_pricing SET cached_input_price = 0.004000 WHERE model_name = 'pa/doubao-seed-1.6-flash';
UPDATE model_pricing SET cached_input_price = 0.009000 WHERE model_name = 'pa/doubao-1-5-pro-32k-250115';
UPDATE model_pricing SET cached_input_price = 0.009000 WHERE model_name = 'pa/doubao-1.5-pro-32k-character-250715';

COMMIT;
