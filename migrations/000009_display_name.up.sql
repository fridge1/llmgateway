-- Add display_name column to models table for showing original model names

BEGIN;

ALTER TABLE models ADD COLUMN display_name VARCHAR(200) NOT NULL DEFAULT '';

-- Populate display_name from the original model name mapping (xlsx 模型列表)
-- OpenAI 系列
UPDATE models SET display_name = 'gpt-5.4-pro' WHERE name = 'pa/gpt-5.4-pro';
UPDATE models SET display_name = 'gpt-5.4' WHERE name = 'pa/gpt-5.4';
UPDATE models SET display_name = 'gpt-5.3-chat-latest' WHERE name = 'pa/gpt-5.3-chat-latest';
UPDATE models SET display_name = 'gpt-5.3-codex' WHERE name = 'pa/gpt-5.3-codex';
UPDATE models SET display_name = 'gpt-5.2-codex' WHERE name = 'pa/gpt-5.2-codex';
UPDATE models SET display_name = 'gpt-5.2' WHERE name = 'pa/gpt-5.2';
UPDATE models SET display_name = 'gpt-5.2-chat-latest' WHERE name = 'pa/gpt-5.2-chat-latest';
UPDATE models SET display_name = 'gpt-5.2-pro' WHERE name = 'pa/gpt-5.2-pro';
UPDATE models SET display_name = 'gpt-5.1-codex-max' WHERE name = 'pa/gpt-5.1-codex-max';
UPDATE models SET display_name = 'gpt-5.1-codex-mini' WHERE name = 'pa/gpt-5.1-codex-mini';
UPDATE models SET display_name = 'gpt-5.1' WHERE name = 'pa/gpt-5.1';
UPDATE models SET display_name = 'gpt-5.1-chat-latest' WHERE name = 'pa/gpt-5.1-chat-latest';
UPDATE models SET display_name = 'gpt-5.1-codex' WHERE name = 'pa/gpt-5.1-codex';
UPDATE models SET display_name = 'gpt-5-pro' WHERE name = 'pa/gpt-5-pro';
UPDATE models SET display_name = 'gpt-5-codex' WHERE name = 'pa/gpt-5-codex';
UPDATE models SET display_name = 'gpt-5' WHERE name = 'pa/gpt-5';
UPDATE models SET display_name = 'gpt-5-mini' WHERE name = 'pa/gpt-5-mini';
UPDATE models SET display_name = 'gpt-5-nano' WHERE name = 'pa/gpt-5-nano';
UPDATE models SET display_name = 'gpt-5-chat-latest' WHERE name = 'pa/gpt-5-chat-latest';
UPDATE models SET display_name = 'gpt-4.1' WHERE name = 'pa/gt-4.1';
UPDATE models SET display_name = 'gpt-4.1-nano' WHERE name = 'pa/gt-4.1-n';
UPDATE models SET display_name = 'gpt-4.1-mini' WHERE name = 'pa/gt-4.1-m';
UPDATE models SET display_name = 'gpt-4o' WHERE name = 'pa/gt-4p';
UPDATE models SET display_name = 'gpt-4o-mini' WHERE name = 'pa/gt-4p-m';
UPDATE models SET display_name = 'o1' WHERE name = 'pa/p1';
UPDATE models SET display_name = 'o1-mini' WHERE name = 'pa/p1-m';
UPDATE models SET display_name = 'o3-mini' WHERE name = 'pa/p3-m';
UPDATE models SET display_name = 'o3' WHERE name = 'pa/p3';
UPDATE models SET display_name = 'o4-mini' WHERE name = 'pa/o4-mini';
UPDATE models SET display_name = 'text-embedding-3-large' WHERE name = 'pa/text-embedding-3-large';

-- Anthropic 系列
UPDATE models SET display_name = 'claude-sonnet-4-6' WHERE name = 'pa/claude-sonnet-4-6';
UPDATE models SET display_name = 'claude-opus-4-6' WHERE name = 'pa/claude-opus-4-6';
UPDATE models SET display_name = 'claude-opus-4-5-20251101' WHERE name = 'pa/claude-opus-4-5-20251101';
UPDATE models SET display_name = 'claude-sonnet-4-5-20250929' WHERE name = 'pa/claude-sonnet-4-5-20250929';
UPDATE models SET display_name = 'claude-sonnet-4-5-20250929 (1M)' WHERE name = 'pa/claude-sonnet-4-5-20250929-1m';
UPDATE models SET display_name = 'claude-haiku-4-5-20251001' WHERE name = 'pa/claude-haiku-4-5-20251001';
UPDATE models SET display_name = 'claude-opus-4-1-20250805' WHERE name = 'pa/claude-opus-4-1-20250805';
UPDATE models SET display_name = 'claude-sonnet-4-20250514' WHERE name = 'pa/cd-st-4-20250514';
UPDATE models SET display_name = 'claude-opus-4-20250514' WHERE name = 'pa/cd-op-4-20250514';
UPDATE models SET display_name = 'claude-3-7-sonnet-20250219' WHERE name = 'pa/cd-3-7-st-20250219';
UPDATE models SET display_name = 'claude-3-5-sonnet-20241022' WHERE name = 'pa/cd-3-5-st-20241022';
UPDATE models SET display_name = 'claude-3-5-haiku-20241022' WHERE name = 'pa/cd-3-5-hk-20241022';
UPDATE models SET display_name = 'claude-3-haiku-20240307' WHERE name = 'pa/cd-3-hk-20240307';

-- Google 系列
UPDATE models SET display_name = 'gemini-2.5-flash' WHERE name = 'pa/gmn-2.5-fls';
UPDATE models SET display_name = 'gemini-2.5-pro' WHERE name = 'pa/gmn-2.5-pr';
UPDATE models SET display_name = 'gemini-2.5-flash-lite' WHERE name = 'pa/gmn-2.5-fls-lt';
UPDATE models SET display_name = 'gemini-2.0-flash' WHERE name = 'pa/gmn-2.0-fls-20250609';
UPDATE models SET display_name = 'gemini-2.0-flash-lite' WHERE name = 'pa/gmn-2.0-fls-lt';
UPDATE models SET display_name = 'gemini-2.5-flash-preview-05-20' WHERE name = 'pa/gmn-2.5-fls-pw-05-20';
UPDATE models SET display_name = 'gemini-2.5-pro-preview-06-05' WHERE name = 'pa/gmn-2.5-pr-pw-06-05';
UPDATE models SET display_name = 'gemini-2.5-flash-lite-preview-06-17' WHERE name = 'pa/gmn-2.5-fls-lt-pw-06-17';
UPDATE models SET display_name = 'gemini-2.5-flash-lite-preview-09-2025' WHERE name = 'pa/gemini-2.5-flash-lite-preview';
UPDATE models SET display_name = 'gemini-3-pro-preview' WHERE name = 'pa/gemini-3-pro-preview';
UPDATE models SET display_name = 'gemini-3-flash-preview' WHERE name = 'pa/gemini-3-flash-preview';
UPDATE models SET display_name = 'gemini-3.1-pro-preview' WHERE name = 'pa/gemini-3.1-pro-preview';
UPDATE models SET display_name = 'gemini-3.1-flash-lite-preview' WHERE name = 'pa/gemini-3.1-flash-lite-preview';

-- Grok 系列
UPDATE models SET display_name = 'grok-4-1-fast-non-reasoning' WHERE name = 'pa/grok-4-1-fast-non-reasoning';
UPDATE models SET display_name = 'grok-4-1-fast-reasoning' WHERE name = 'pa/grok-4-1-fast-reasoning';
UPDATE models SET display_name = 'grok-4' WHERE name = 'pa/grk-4';
UPDATE models SET display_name = 'grok-4-fast-reasoning' WHERE name = 'pa/grok-4-fast-reasoning';
UPDATE models SET display_name = 'grok-4-fast-non-reasoning' WHERE name = 'pa/grok-4-fast-non-reasoning';
UPDATE models SET display_name = 'grok-code-fast-1' WHERE name = 'pa/grok-code-fast-1';
UPDATE models SET display_name = 'grok-3' WHERE name = 'pa/grk-3';
UPDATE models SET display_name = 'grok-3-mini' WHERE name = 'pa/grok-3-mini';

-- 豆包 系列
UPDATE models SET display_name = 'doubao-seed-1-8-251228' WHERE name = 'pa/doubao-seed-1-8-251228';
UPDATE models SET display_name = 'doubao-seed-1.6' WHERE name = 'pa/doubao-seed-1.6';
UPDATE models SET display_name = 'doubao-seed-1.6-thinking' WHERE name = 'pa/doubao-seed-1.6-thinking';
UPDATE models SET display_name = 'doubao-seed-1.6-flash' WHERE name = 'pa/doubao-seed-1.6-flash';
UPDATE models SET display_name = 'doubao-1-5-pro-32k-250115' WHERE name = 'pa/doubao-1-5-pro-32k-250115';
UPDATE models SET display_name = 'doubao-1.5-pro-32k-character-250715' WHERE name = 'pa/doubao-1.5-pro-32k-character-250715';

COMMIT;
