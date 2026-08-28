-- Seed model_pricing with official prices × 0.8 (20% discount)
-- Prices are per 1,000,000 tokens in USD
-- Doubao prices converted from CNY to USD at ~¥7.2/$1 then ×0.8

BEGIN;

-- ============================================================
-- OpenAI 系列 (official × 0.8)
-- ============================================================
INSERT INTO model_pricing (model_name, input_price, output_price, is_active) VALUES
-- GPT-5.4 系列
('pa/gpt-5.4-pro',           24.000000, 144.000000, true),
('pa/gpt-5.4',                2.000000,  12.000000, true),
-- GPT-5.3 系列
('pa/gpt-5.3-chat-latest',    2.000000,  12.000000, true),
('pa/gpt-5.3-codex',          1.000000,   8.000000, true),
-- GPT-5.2 系列
('pa/gpt-5.2-codex',          1.000000,   8.000000, true),
('pa/gpt-5.2',                1.000000,   8.000000, true),
('pa/gpt-5.2-chat-latest',    1.000000,   8.000000, true),
('pa/gpt-5.2-pro',           24.000000, 144.000000, true),
-- GPT-5.1 系列
('pa/gpt-5.1-codex-max',      1.000000,   8.000000, true),
('pa/gpt-5.1-codex-mini',     0.200000,   1.600000, true),
('pa/gpt-5.1',                1.000000,   8.000000, true),
('pa/gpt-5.1-chat-latest',    1.000000,   8.000000, true),
('pa/gpt-5.1-codex',          1.000000,   8.000000, true),
-- GPT-5 系列
('pa/gpt-5-pro',             24.000000, 144.000000, true),
('pa/gpt-5-codex',            1.000000,   8.000000, true),
('pa/gpt-5',                  1.000000,   8.000000, true),
('pa/gpt-5-mini',             0.200000,   1.600000, true),
('pa/gpt-5-nano',             0.040000,   0.320000, true),
('pa/gpt-5-chat-latest',      1.000000,   8.000000, true),
-- GPT-4.1 系列
('pa/gt-4.1',                 1.600000,   6.400000, true),
('pa/gt-4.1-n',               0.080000,   0.320000, true),
('pa/gt-4.1-m',               0.320000,   1.280000, true),
-- GPT-4o 系列
('pa/gt-4p',                  2.000000,   8.000000, true),
('pa/gt-4p-m',                0.120000,   0.480000, true),
-- o 系列
('pa/p1',                    12.000000,  48.000000, true),
('pa/p1-m',                   2.400000,   9.600000, true),
('pa/p3-m',                   0.880000,   3.520000, true),
('pa/p3',                     1.600000,   6.400000, true),
('pa/o4-mini',                1.600000,   6.400000, true),
-- Embedding
('pa/text-embedding-3-large', 0.104000,   0.000000, true),

-- ============================================================
-- Anthropic 系列 (official × 0.8)
-- ============================================================
('pa/claude-sonnet-4-6',              2.400000,  12.000000, true),
('pa/claude-opus-4-6',                4.000000,  20.000000, true),
('pa/claude-opus-4-5-20251101',       4.000000,  20.000000, true),
('pa/claude-sonnet-4-5-20250929',     2.400000,  12.000000, true),
('pa/claude-sonnet-4-5-20250929-1m',  2.400000,  12.000000, true),
('pa/claude-haiku-4-5-20251001',      0.800000,   4.000000, true),
('pa/claude-opus-4-1-20250805',      12.000000,  60.000000, true),
('pa/cd-st-4-20250514',              2.400000,  12.000000, true),
('pa/cd-op-4-20250514',             12.000000,  60.000000, true),
('pa/cd-3-7-st-20250219',            2.400000,  12.000000, true),
('pa/cd-3-5-st-20241022',            2.400000,  12.000000, true),
('pa/cd-3-5-hk-20241022',            0.640000,   3.200000, true),
('pa/cd-3-hk-20240307',              0.200000,   1.000000, true),

-- ============================================================
-- Google Gemini 系列 (official × 0.8)
-- ============================================================
('pa/gmn-2.5-fls',                       0.240000,   2.000000, true),
('pa/gmn-2.5-pr',                        1.000000,   8.000000, true),
('pa/gmn-2.5-fls-lt',                    0.080000,   0.320000, true),
('pa/gmn-2.0-fls-20250609',              0.080000,   0.320000, true),
('pa/gmn-2.0-fls-lt',                    0.080000,   0.320000, true),
('pa/gmn-2.5-fls-pw-05-20',             0.240000,   2.000000, true),
('pa/gmn-2.5-pr-pw-06-05',              1.000000,   8.000000, true),
('pa/gmn-2.5-fls-lt-pw-06-17',          0.080000,   0.320000, true),
('pa/gemini-2.5-flash-lite-preview',     0.080000,   0.320000, true),
('pa/gemini-3-pro-preview',              1.600000,   9.600000, true),
('pa/gemini-3-flash-preview',            0.400000,   0.800000, true),
('pa/gemini-3.1-pro-preview',            1.600000,   9.600000, true),
('pa/gemini-3.1-flash-lite-preview',     0.080000,   0.320000, true),

-- ============================================================
-- Grok / xAI 系列 (official × 0.8)
-- ============================================================
('pa/grok-4-1-fast-non-reasoning',  0.160000,   0.800000, true),
('pa/grok-4-1-fast-reasoning',      0.160000,   0.800000, true),
('pa/grk-4',                        2.400000,  12.000000, true),
('pa/grok-4-fast-reasoning',        0.160000,   0.800000, true),
('pa/grok-4-fast-non-reasoning',    0.160000,   0.800000, true),
('pa/grok-code-fast-1',             0.160000,   0.400000, true),
('pa/grk-3',                        2.400000,  12.000000, true),
('pa/grok-3-mini',                  0.240000,   0.400000, true),

-- ============================================================
-- 豆包 / Volcengine 系列 (CNY → USD at ¥7.2/$1, then × 0.8)
-- doubao-seed-1.6:  ¥0.80/¥2.00 → $0.111/$0.278 → ×0.8 = $0.089/$0.222
-- doubao-seed-flash: ¥0.40/¥1.00 → $0.056/$0.139 → ×0.8 = $0.044/$0.111
-- ============================================================
('pa/doubao-seed-1-8-251228',                0.089000,   0.222000, true),
('pa/doubao-seed-1.6',                       0.089000,   0.222000, true),
('pa/doubao-seed-1.6-thinking',              0.089000,   0.222000, true),
('pa/doubao-seed-1.6-flash',                 0.044000,   0.111000, true),
('pa/doubao-1-5-pro-32k-250115',             0.089000,   0.222000, true),
('pa/doubao-1.5-pro-32k-character-250715',   0.089000,   0.222000, true)

ON CONFLICT (model_name) DO UPDATE SET
    input_price  = EXCLUDED.input_price,
    output_price = EXCLUDED.output_price,
    is_active    = EXCLUDED.is_active,
    updated_at   = NOW();

COMMIT;
