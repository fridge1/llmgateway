-- Expand category column to TEXT and update all models with multi-category classifications.
-- Categories: text, multimodal, reasoning, embedding (comma-separated)

BEGIN;

ALTER TABLE models ALTER COLUMN category TYPE TEXT;

-- ============================================================
-- Text-only models (纯文本对话/代码)
-- ============================================================
UPDATE models SET category = 'text' WHERE name IN (
  'pa/gpt-5-nano',
  'pa/gpt-5-mini',
  'pa/gt-4.1-n',
  'pa/gt-4p-m',
  'pa/gpt-5-codex',
  'pa/gpt-5-chat-latest',
  'pa/gpt-5.1-codex-mini',
  'pa/gpt-5.1-codex',
  'pa/gpt-5.1-codex-max',
  'pa/gpt-5.1',
  'pa/gpt-5.1-chat-latest',
  'pa/gpt-5.2-codex',
  'pa/gpt-5.2',
  'pa/gpt-5.2-chat-latest',
  'pa/gpt-5.3-codex',
  'pa/gpt-5.3-chat-latest',
  'pa/grok-3-mini',
  'pa/grok-4-1-fast-non-reasoning',
  'pa/grok-4-fast-non-reasoning',
  'pa/grok-code-fast-1',
  'pa/cd-3-hk-20240307',
  'pa/gmn-2.0-fls-lt',
  'pa/gmn-2.5-fls-lt',
  'pa/gmn-2.5-fls-lt-pw-06-17',
  'pa/gemini-2.5-flash-lite-preview',
  'pa/gemini-3.1-flash-lite-preview',
  'pa/doubao-1-5-pro-32k-250115',
  'pa/doubao-1.5-pro-32k-character-250715',
  'pa/doubao-seed-1.6-flash'
);

-- ============================================================
-- Multimodal-only models (多模态，不含推理)
-- ============================================================
UPDATE models SET category = 'multimodal' WHERE name IN (
  'pa/gt-4p',
  'pa/gt-4.1',
  'pa/gt-4.1-m',
  'pa/gpt-5',
  'pa/gpt-5-pro',
  'pa/gpt-5.2-pro',
  'pa/gpt-5.4',
  'pa/gpt-5.4-pro',
  'pa/cd-3-5-hk-20241022',
  'pa/cd-3-5-st-20241022',
  'pa/cd-st-4-20250514',
  'pa/cd-op-4-20250514',
  'pa/claude-haiku-4-5-20251001',
  'pa/claude-opus-4-5-20251101',
  'pa/claude-opus-4-1-20250805',
  'pa/gmn-2.0-fls-20250609',
  'pa/gmn-2.5-fls-pw-05-20',
  'pa/gmn-2.5-pr-pw-06-05',
  'pa/gemini-3-flash-preview',
  'pa/gemini-3-pro-preview',
  'pa/gemini-3.1-pro-preview',
  'pa/grk-3',
  'pa/grk-4',
  'pa/doubao-seed-1.6',
  'pa/doubao-seed-1-8-251228'
);

-- ============================================================
-- Multimodal + Reasoning models (多��态 + 推理)
-- ============================================================
UPDATE models SET category = 'multimodal,reasoning' WHERE name IN (
  'pa/cd-3-7-st-20250219',
  'pa/claude-sonnet-4-5-20250929',
  'pa/claude-sonnet-4-5-20250929-1m',
  'pa/claude-sonnet-4-6',
  'pa/claude-opus-4-6',
  'pa/gmn-2.5-fls',
  'pa/gmn-2.5-pr',
  'pa/doubao-seed-1.6-thinking'
);

-- ============================================================
-- Reasoning-only models (纯推理)
-- ============================================================
UPDATE models SET category = 'reasoning' WHERE name IN (
  'pa/p1',
  'pa/p1-m',
  'pa/p3',
  'pa/p3-m',
  'pa/o4-mini',
  'pa/grok-4-1-fast-reasoning',
  'pa/grok-4-fast-reasoning'
);

-- ============================================================
-- Embedding models (向量嵌入)
-- ============================================================
UPDATE models SET category = 'embedding' WHERE name IN (
  'pa/text-embedding-3-large'
);

COMMIT;
