-- Seed models from 模型列表.xlsx
-- api_key 统一使用提供的 key，base_url 后续按平台单独配置

BEGIN;

-- ============================================================
-- OpenAI 系列
-- ============================================================
WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.4-pro') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.4-pro', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.4') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.4', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.3-chat-latest') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.3-chat-latest', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.3-codex') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.3-codex', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.2-codex') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.2-codex', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.2') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.2', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.2-chat-latest') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.2-chat-latest', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.2-pro') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.2-pro', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.1-codex-max') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.1-codex-max', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.1-codex-mini') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.1-codex-mini', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.1') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.1', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.1-chat-latest') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.1-chat-latest', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5.1-codex') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5.1-codex', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5-pro') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5-pro', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5-codex') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5-codex', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5-mini') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5-mini', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5-nano') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5-nano', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gpt-5-chat-latest') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-5-chat-latest', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gt-4.1') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-4.1', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gt-4.1-n') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-4.1-nano', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gt-4.1-m') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-4.1-mini', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gt-4p') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-4o', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gt-4p-m') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'gpt-4o-mini', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/p1') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'o1', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/p1-m') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'o1-mini', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/p3-m') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'o3-mini', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/p3') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'o3', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/o4-mini') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'o4-mini', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/text-embedding-3-large') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'openai', 'https://api.openai.com/v1', 'sk-your-upstream-api-key', 'text-embedding-3-large', 1 FROM m;

-- ============================================================
-- Anthropic 系列
-- ============================================================
WITH m AS (
    INSERT INTO models (name) VALUES ('pa/claude-sonnet-4-6') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-sonnet-4-6', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/claude-opus-4-6') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-opus-4-6', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/claude-opus-4-5-20251101') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-opus-4-5-20251101', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/claude-sonnet-4-5-20250929') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-sonnet-4-5-20250929', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/claude-haiku-4-5-20251001') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-haiku-4-5-20251001', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/claude-opus-4-1-20250805') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-opus-4-1-20250805', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/cd-st-4-20250514') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-sonnet-4-20250514', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/cd-op-4-20250514') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-opus-4-20250514', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/cd-3-7-st-20250219') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-3-7-sonnet-20250219', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/cd-3-5-st-20241022') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-3-5-sonnet-20241022', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/cd-3-5-hk-20241022') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-3-5-haiku-20241022', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/cd-3-hk-20240307') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'anthropic', 'https://api.anthropic.com/v1', 'sk-your-upstream-api-key', 'claude-3-haiku-20240307', 1 FROM m;

-- ============================================================
-- Google 系列
-- ============================================================
WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gmn-2.5-fls') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-2.5-flash', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gmn-2.5-pr') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-2.5-pro', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gmn-2.5-fls-lt') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-2.5-flash-lite', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gmn-2.0-fls-20250609') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-2.0-flash', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gmn-2.0-fls-lt') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-2.0-flash-lite', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gmn-2.5-fls-pw-05-20') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-2.5-flash-preview-05-20', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gmn-2.5-pr-pw-06-05') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-2.5-pro-preview-06-05', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gmn-2.5-fls-lt-pw-06-17') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-2.5-flash-lite-preview-06-17', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gemini-2.5-flash-lite-preview') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-2.5-flash-lite-preview-09-2025', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gemini-3-pro-preview') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-3-pro-preview', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gemini-3-flash-preview') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-3-flash-preview', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gemini-3.1-pro-preview') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-3.1-pro-preview', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/gemini-3.1-flash-lite-preview') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'google', 'https://generativelanguage.googleapis.com/v1beta', 'sk-your-upstream-api-key', 'gemini-3.1-flash-lite-preview', 1 FROM m;

-- ============================================================
-- Grok 系列 (xAI)
-- ============================================================
WITH m AS (
    INSERT INTO models (name) VALUES ('pa/grok-4-1-fast-non-reasoning') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'xai', 'https://api.x.ai/v1', 'sk-your-upstream-api-key', 'grok-4-1-fast-non-reasoning', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/grok-4-1-fast-reasoning') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'xai', 'https://api.x.ai/v1', 'sk-your-upstream-api-key', 'grok-4-1-fast-reasoning', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/grk-4') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'xai', 'https://api.x.ai/v1', 'sk-your-upstream-api-key', 'grok-4', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/grok-4-fast-reasoning') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'xai', 'https://api.x.ai/v1', 'sk-your-upstream-api-key', 'grok-4-fast-reasoning', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/grok-4-fast-non-reasoning') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'xai', 'https://api.x.ai/v1', 'sk-your-upstream-api-key', 'grok-4-fast-non-reasoning', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/grok-code-fast-1') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'xai', 'https://api.x.ai/v1', 'sk-your-upstream-api-key', 'grok-code-fast-1', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/grk-3') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'xai', 'https://api.x.ai/v1', 'sk-your-upstream-api-key', 'grok-3', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/grok-3-mini') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'xai', 'https://api.x.ai/v1', 'sk-your-upstream-api-key', 'grok-3-mini', 1 FROM m;

-- ============================================================
-- 豆包 系列 (Volcengine)
-- ============================================================
WITH m AS (
    INSERT INTO models (name) VALUES ('pa/doubao-seed-1-8-251228') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'volcengine', 'https://ark.cn-beijing.volces.com/api/v3', 'sk-your-upstream-api-key', 'doubao-seed-1-8-251228', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/doubao-seed-1.6') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'volcengine', 'https://ark.cn-beijing.volces.com/api/v3', 'sk-your-upstream-api-key', 'doubao-seed-1.6', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/doubao-seed-1.6-thinking') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'volcengine', 'https://ark.cn-beijing.volces.com/api/v3', 'sk-your-upstream-api-key', 'doubao-seed-1.6-thinking', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/doubao-seed-1.6-flash') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'volcengine', 'https://ark.cn-beijing.volces.com/api/v3', 'sk-your-upstream-api-key', 'doubao-seed-1.6-flash', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/doubao-1-5-pro-32k-250115') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'volcengine', 'https://ark.cn-beijing.volces.com/api/v3', 'sk-your-upstream-api-key', 'doubao-1-5-pro-32k-250115', 1 FROM m;

WITH m AS (
    INSERT INTO models (name) VALUES ('pa/doubao-1.5-pro-32k-character-250715') RETURNING id
) INSERT INTO upstreams (model_id, provider, base_url, api_key, model_override, weight) SELECT id, 'volcengine', 'https://ark.cn-beijing.volces.com/api/v3', 'sk-your-upstream-api-key', 'doubao-1.5-pro-32k-character-250715', 1 FROM m;

COMMIT;