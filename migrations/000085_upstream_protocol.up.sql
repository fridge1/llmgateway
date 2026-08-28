-- 为 upstreams 表添加独立的 protocol 字段，将"路由协议"与"UI 分组用 provider"解耦。
-- 空字符串表示沿用旧逻辑（按 provider 推断），非空时强制按 protocol 路由。
-- 取值约定：'openai' / 'anthropic' / 'gemini'。
ALTER TABLE upstreams
ADD COLUMN IF NOT EXISTS protocol VARCHAR(20) NOT NULL DEFAULT '';
