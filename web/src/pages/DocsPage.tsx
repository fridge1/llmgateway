import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeft, Copy, CheckCircle, Zap, ChevronRight, BookOpen, PlayCircle } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import BeianBar from "@/components/BeianBar";
import { Seo } from "@/components/Seo";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const menuItems = [
  { key: "quickstart", label: "快速开始" },
  { key: "claude-code", label: "Claude Code" },
  { key: "claude-tips", label: "Claude 省钱技巧" },
  { key: "claude-models", label: "Claude 模型选择指南" },
  { key: "api-overview", label: "API 概览" },
  { key: "auth", label: "认证方式" },
  { key: "compatibility", label: "兼容性矩阵" },
  { key: "chat-completions", label: "Chat Completions" },
  { key: "messages-api", label: "Messages API" },
  { key: "responses-api", label: "Responses API" },
  { key: "image-tasks", label: "图像任务（异步）" },
  { key: "models", label: "模型列表" },
  { key: "balance", label: "查询余额" },
  { key: "cc-switch", label: "CC Switch" },
  { key: "codex-cli", label: "Codex CLI" },
  { key: "cursor", label: "Cursor" },
  { key: "copilot-cli", label: "Copilot CLI" },
  { key: "trae", label: "Trae" },
  { key: "opencode", label: "OpenCode" },
  { key: "gemini-api", label: "Gemini API" },
  { key: "streaming", label: "流式与超时" },
  { key: "errors", label: "错误处理" },
];

interface CodeBlockProps {
  lang: string;
  code: string;
  blockId: string;
  copiedId: string;
  onCopy: (code: string, id: string) => void;
}

const TOS_IMG_BASE = "https://your-tos-bucket.tos-cn-beijing.volces.com/ConfigurationTutorial";

interface DocImageProps {
  src: string;
  alt: string;
  caption?: string;
}

const DocImage = ({ src, alt, caption }: DocImageProps) => (
  <figure className="mb-5">
    <a href={src} target="_blank" rel="noopener noreferrer" className="block group">
      <img
        src={src}
        alt={alt}
        loading="lazy"
        className="w-full max-w-3xl rounded-xl border border-border bg-muted/20 transition-shadow group-hover:shadow-md"
      />
    </a>
    {caption && (
      <figcaption className="mt-2 text-xs text-muted-foreground">{caption}</figcaption>
    )}
  </figure>
);

const CodeBlock = ({ lang, code, blockId, copiedId, onCopy }: CodeBlockProps) => (
  <div className="code-block mb-5 rounded-xl overflow-hidden">
    <div
      className="flex items-center justify-between px-4 py-2.5 border-b"
      style={{ borderColor: "rgba(51,65,85,0.5)" }}
    >
      <span className="text-xs font-semibold tracking-wider" style={{ color: "rgba(100,116,139,1)" }}>
        {lang}
      </span>
      <button
        onClick={() => onCopy(code, blockId)}
        className="flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-lg transition-all duration-200 hover:bg-white/10"
        style={{ color: copiedId === blockId ? "rgba(16,185,129,1)" : "rgba(100,116,139,1)" }}
      >
        {copiedId === blockId ? <CheckCircle size={12} /> : <Copy size={12} />}
        {copiedId === blockId ? "已复制" : "复制"}
      </button>
    </div>
    <pre className="px-5 py-4 text-sm overflow-x-auto leading-relaxed" style={{ color: "rgba(226,232,240,1)" }}>
      <code>{code}</code>
    </pre>
  </div>
);

const DocsPage = () => {
  const navigate = useNavigate();
  const auth = useAuth();
  const [activeSection, setActiveSection] = useState("quickstart");
  const [copiedId, setCopiedId] = useState("");

  const handleCopy = (code: string, id: string) => {
    navigator.clipboard.writeText(code).then(() => {
      setCopiedId(id);
      setTimeout(() => setCopiedId(""), 2000);
    });
  };

  // --- API Overview examples ---

  const chatCompletionsCurl = `curl -X POST https://your-domain.com/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-6",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello!"}
    ],
    "stream": false
  }'`;

  const chatCompletionsResponse = `{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1713000000,
  "model": "claude-sonnet-4-6",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 10,
    "total_tokens": 30
  }
}`;

  const chatCompletionsStreamCurl = `curl -X POST https://your-domain.com/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-6",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "stream": true
  }'`;

  const messagesApiCurl = `curl -X POST https://your-domain.com/v1/messages \\
  -H "x-api-key: YOUR_API_KEY" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-6",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'`;

  const messagesApiResponse = `{
  "id": "msg_abc123",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "Hello! How can I help you today?"
    }
  ],
  "model": "claude-sonnet-4-6",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 10,
    "output_tokens": 12
  }
}`;

  const responsesApiCurl = `curl -X POST https://your-domain.com/v1/responses \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-5.4",
    "input": [
      {"role": "user", "content": "Hello!"}
    ]
  }'`;

  const responsesApiResponse = `{
  "id": "resp_abc123",
  "object": "response",
  "status": "completed",
  "model": "gpt-5.4",
  "output": [
    {
      "type": "message",
      "id": "msg_abc123",
      "role": "assistant",
      "content": [
        {"type": "output_text", "text": "Hello! How can I help you?"}
      ],
      "status": "completed"
    }
  ],
  "usage": {
    "input_tokens": 10,
    "output_tokens": 12,
    "total_tokens": 22
  }
}`;

  const imageTaskSubmitCurl = `# 提交生成任务（立即返回任务 ID，不阻塞）
curl -X POST https://your-domain.com/v1/images/tasks \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "a red apple on a wooden table, studio lighting",
    "size": "1024x1024",
    "n": 2
  }'`;

  const imageTaskSubmitResponse = `{
  "id": 123,
  "status": "pending"
}`;

  const imageTaskEditCurl = `# 提交编辑任务（输入图用 URL 或 base64，可混用，最多 4 张、单张 ≤ 25MB）
curl -X POST https://your-domain.com/v1/images/tasks/edits \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "make the background pure white",
    "size": "1024x1024",
    "n": 1,
    "image_urls": ["https://example.com/cat.png"],
    "image_base64s": [],
    "mask_base64": ""
  }'`;

  const imageTaskPollCurl = `# 轮询任务状态（建议 2-5 秒一次，直到 completed / failed）
curl https://your-domain.com/v1/images/tasks/123 \\
  -H "Authorization: Bearer YOUR_API_KEY"`;

  const imageTaskPollResponse = `{
  "id": 123,
  "type": "generate",
  "status": "completed",
  "model": "gpt-image-2",
  "prompt": "a red apple on a wooden table, studio lighting",
  "size": "1024x1024",
  "image_count": 2,
  "result_urls": [
    "https://your-tos-bucket.tos-cn-beijing.volces.com/xxx-1.png",
    "https://your-tos-bucket.tos-cn-beijing.volces.com/xxx-2.png"
  ],
  "cost": 0.16,
  "created_at": "2026-06-21T12:00:00Z",
  "completed_at": "2026-06-21T12:01:30Z"
}`;

  const imageTaskDeleteCurl = `# 删除任务（仅 pending / completed / failed 可删；处理中返回 409）
curl -X DELETE https://your-domain.com/v1/images/tasks/123 \\
  -H "Authorization: Bearer YOUR_API_KEY"
# 成功返回 204 No Content`;

  const modelsApiCurl = `curl https://your-domain.com/v1/models \\
  -H "Authorization: Bearer YOUR_API_KEY"`;
  const modelsApiResponse = `{
  "object": "list",
  "data": [
    {"id": "claude-sonnet-4-6", "object": "model", "created": 1712973600, "owned_by": "llm-gateway"},
    {"id": "gpt-5.4", "object": "model", "created": 1712973600, "owned_by": "llm-gateway"},
    ...
  ]
}`;

  const balanceApiCurl = `curl https://your-domain.com/v1/balance \\
  -H "Authorization: Bearer YOUR_API_KEY"`;
  const balanceApiResponse = `{
  "type": "user",
  "balance": 100.50,
  "frozen": 5.00,
  "available": 95.50,
  "currency": "CNY"
}`;

  const geminiApiCurl = `curl -X POST "https://your-domain.com/gemini/v1/models/gemini-2.5-flash:generateContent" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "contents": [
      {"parts": [{"text": "Hello!"}]}
    ]
  }'`;

  const geminiStreamCurl = `curl -X POST "https://your-domain.com/gemini/v1/models/gemini-2.5-flash:streamGenerateContent?alt=sse" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "contents": [
      {"parts": [{"text": "Hello!"}]}
    ]
  }'`;

  const geminiImageCurl = `curl -X POST "https://your-domain.com/gemini/v1beta/models/gemini-3.1-flash-image-preview:generateContent" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "contents": [
      {"parts": [{"text": "画一只可爱的猫咪"}]}
    ],
    "generationConfig": {
      "responseModalities": ["TEXT", "IMAGE"]
    }
  }'`;

  const geminiPythonSdk = `from google import genai

client = genai.Client(
    api_key="YOUR_API_KEY",
    http_options={"api_version": "v1beta",
                  "url": "https://your-domain.com/gemini"},
)

# 文本生成
response = client.models.generate_content(
    model="gemini-2.5-flash",
    contents="Hello!",
)
print(response.text)

# 图像生成（需要 v1beta）
from google.genai import types

response = client.models.generate_content(
    model="gemini-3.1-flash-image-preview",
    contents="画一只可爱的猫咪",
    config=types.GenerateContentConfig(
        response_modalities=["TEXT", "IMAGE"],
    ),
)
for part in response.candidates[0].content.parts:
    if part.inline_data:
        # part.inline_data.data 为图片的 base64 编码
        print(f"图片 MIME: {part.inline_data.mime_type}")
    elif part.text:
        print(part.text)`;

  const claudeCodeEnvCode = `# 设置环境变量（macOS / Linux 写入 ~/.zshrc 或 ~/.bashrc 永久生效）
export ANTHROPIC_BASE_URL="https://your-domain.com"
export ANTHROPIC_AUTH_TOKEN="YOUR_API_KEY"
# ANTHROPIC_MODEL 可设置为网关中任意聊天模型
# 例如：claude-sonnet-4-6, gpt-5.4, gemini-3.1-pro-preview, glm-5.1 等
export ANTHROPIC_MODEL="claude-sonnet-4-6"

# 让 Claude Code 内置档位（Sonnet / Opus / Haiku）映射到网关模型
export ANTHROPIC_DEFAULT_SONNET_MODEL="claude-sonnet-4-6"
export ANTHROPIC_DEFAULT_OPUS_MODEL="claude-opus-4-6"
export ANTHROPIC_DEFAULT_HAIKU_MODEL="claude-haiku-4-5-20251001"`;

  const claudeCodeRunCode = `# 启动 Claude Code
claude

# 或直接在项目目录中启动
cd your-project && claude`;

  const claudeCodeModelsCode = `# 可用模型（根据网关实际配置，完整列表请调用 GET /v1/models）

# Claude 系列
claude-sonnet-4-6           # Sonnet 4.6
claude-opus-4-6             # Opus 4.6
claude-haiku-4-5-20251001   # Haiku 4.5

# OpenAI 系列
gpt-5.4                     # GPT-5.4
gpt-5.4-codex               # GPT-5.4 Codex
gpt-5.4-codex-high          # GPT-5.4 Codex High

# Gemini 系列
gemini-3.1-pro-preview      # Gemini 3.1 Pro Preview

# 智谱 GLM 系列
glm-5.1                     # GLM 5.1`;

  const claudeCodeSettingsCode = `# 方式一：写入 shell 配置文件（推荐，永久生效）
echo 'export ANTHROPIC_BASE_URL="https://your-domain.com"' >> ~/.bashrc
echo 'export ANTHROPIC_AUTH_TOKEN="YOUR_API_KEY"' >> ~/.bashrc
echo 'export ANTHROPIC_MODEL="claude-sonnet-4-6"' >> ~/.bashrc  # 可替换为任意聊天模型

# 让 Claude Code 内置档位（Sonnet / Opus / Haiku）映射到网关模型
echo 'export ANTHROPIC_DEFAULT_SONNET_MODEL="claude-sonnet-4-6"' >> ~/.bashrc
echo 'export ANTHROPIC_DEFAULT_OPUS_MODEL="claude-opus-4-6"' >> ~/.bashrc
echo 'export ANTHROPIC_DEFAULT_HAIKU_MODEL="claude-haiku-4-5-20251001"' >> ~/.bashrc

source ~/.bashrc

# 方式二：在 Claude Code 内切换模型
# 启动后输入 /model 命令即可交互选择模型
# 支持所有网关中的聊天模型，如 gpt-5.4、gemini-3.1-pro-preview 等`;

  // --- Codex CLI examples ---

  // --- CC Switch examples ---

  const ccSwitchInstallCode = `# 方式一：桌面版（推荐）
# 从 GitHub 下载对应平台的安装包：
# https://github.com/farion1231/cc-switch/releases

# 方式二：CLI 版
npm install -g cc-switch-cli`;

  const ccSwitchConfigCode = `# 在 cc-switch 中添加 Provider 时填写以下信息：
#
# Provider 名称:  LLM Gateway（或自定义名称）
# Base URL:       https://your-domain.com
# API Key:        在控制台「API 密钥」页面创建的 sk-xxx 密钥
# 模型列表:       点击「获取模型」自动拉取，或手动输入`;

  const ccSwitchToolsTable = `# 本网关支持以下工具，cc-switch 均可一键切换：
#
# 工具            API 格式                端点
# ─────────────  ─────────────────────  ──────────────────────
# Claude Code    Anthropic Messages     /v1/messages
# Codex CLI      OpenAI Responses       /v1/responses
# Cursor         OpenAI Chat            /v1/chat/completions
# Copilot CLI    Anthropic Messages     /v1/messages  (BYOK type=anthropic)
# Trae           OpenAI Chat            /v1/chat/completions
# OpenCode       OpenAI Chat            /v1/chat/completions
# Windsurf       OpenAI Chat            /v1/chat/completions
# Gemini CLI     Google Gemini          /gemini/v1/`;

  const ccSwitchManualCode = `# 如果不使用 cc-switch，也可以手动配置环境变量：

# Claude Code
export ANTHROPIC_BASE_URL="https://your-domain.com"
export ANTHROPIC_AUTH_TOKEN="sk-your-api-key"
export ANTHROPIC_MODEL="claude-sonnet-4-6"

# Codex CLI
export OPENAI_BASE_URL="https://your-domain.com/v1"
export OPENAI_API_KEY="sk-your-api-key"

# Cursor / Windsurf
# 在设置中将 Base URL 设为 https://your-domain.com/v1
# API Key 填写 sk-your-api-key`;

  // --- Codex CLI code examples ---

  const codexEnvCode = `# 设置环境变量
export OPENAI_BASE_URL="https://your-domain.com/v1"
export OPENAI_API_KEY="YOUR_API_KEY"`;

  const codexConfigToml = `# ~/.codex/config.toml
[model_providers.gateway]
name = "LLM Gateway"
base_url = "https://your-domain.com/v1"
env_key = "OPENAI_API_KEY"

profile = "default"

[profiles.default]
model = "gpt-5.4"
model_provider = "gateway"
wire_api = "chat"`;

  const codexConfigEnv = `# 设置配置文件中引用的环境变量
export OPENAI_API_KEY="YOUR_API_KEY"`;

  const codexAuthJson = `// ~/.codex/auth.json — Codex 启动时需要此文件
{
  "auth_mode": "apikey",
  "OPENAI_API_KEY": "YOUR_API_KEY"
}`;

  const codexModelsCode = `# OpenAI 系列（推荐用于 Codex）
gpt-5.4             # GPT-5.4
gpt-5.4-codex       # GPT-5.4 Codex — 代码专用
gpt-5.4-codex-high  # GPT-5.4 Codex High

# Claude 系列（同样支持）
claude-sonnet-4-6   # Sonnet 4.6
claude-opus-4-6     # Opus 4.6

# 其他模型
gemini-3.1-pro-preview  # Gemini 3.1 Pro Preview
glm-5.1                 # GLM 5.1`;

  const codexRunCode = `# 使用默认模型启动
codex

# 指定模型启动
codex --model gpt-5.4

# 直接提问
codex --model gpt-5.4 "explain this codebase"`;

  const codexPersistCode = `# 写入 shell 配置文件（永久生效）
echo 'export OPENAI_BASE_URL="https://your-domain.com/v1"' >> ~/.bashrc
echo 'export OPENAI_API_KEY="YOUR_API_KEY"' >> ~/.bashrc
source ~/.bashrc`;

  // --- Cursor examples ---

  const cursorOpenAISettingsCode = `// Cursor Settings → Models → OpenAI API Key
// 1. 填入你的网关 API Key
// 2. 勾选 "Override OpenAI Base URL"
// 3. 填入网关地址（带 /v1）：
https://your-domain.com/v1
// 4. 点击 Verify，绿色勾即可`;

  const cursorAddModelCode = `// Cursor 添加自定义模型时，模型名【必须】带 gw/ 前缀
// 否则 Cursor 会按官方 OpenAI 模型名拦截/改写，导致 Verify 失败或调不到对应模型

// ✅ 正确
gw/claude-sonnet-4-6
gw/gpt-5.4-codex

// ❌ 错误（会被 Cursor 拦截）
claude-sonnet-4-6
gpt-5.4-codex`;

  const cursorModelsCode = `# Cursor 中所有模型名都要带 gw/ 前缀

# OpenAI 系列
gw/gpt-5.4
gw/gpt-5.4-codex
gw/gpt-5.4-codex-high

# Claude 系列（通过 OpenAI 协议透传，由网关自动转换）
gw/claude-sonnet-4-6
gw/claude-opus-4-6

# 其他
gw/gemini-3.1-pro-preview
gw/glm-5.1`;

  // --- Copilot CLI examples ---

  const copilotEnvCode = `# 推荐配置：BYOK 走 Anthropic 协议（兼容性最佳）
export COPILOT_PROVIDER_TYPE="anthropic"
export COPILOT_PROVIDER_BASE_URL="https://your-domain.com"
export COPILOT_PROVIDER_API_KEY="YOUR_API_KEY"
export COPILOT_MODEL="claude-sonnet-4-6"

# 上下文与输出长度上限（按所选模型实际能力调整）
export COPILOT_PROVIDER_MAX_PROMPT_TOKENS="200000"
export COPILOT_PROVIDER_MAX_OUTPUT_TOKENS="8192"`;

  const copilotPersistCode = `# 写入 shell 配置文件（永久生效）
{
  echo 'export COPILOT_PROVIDER_TYPE="anthropic"'
  echo 'export COPILOT_PROVIDER_BASE_URL="https://your-domain.com"'
  echo 'export COPILOT_PROVIDER_API_KEY="YOUR_API_KEY"'
  echo 'export COPILOT_MODEL="claude-sonnet-4-6"'
  echo 'export COPILOT_PROVIDER_MAX_PROMPT_TOKENS="200000"'
  echo 'export COPILOT_PROVIDER_MAX_OUTPUT_TOKENS="8192"'
} >> ~/.bashrc
source ~/.bashrc`;

  const copilotOpenAITypeCode = `# 备选方案：BYOK 走 OpenAI 协议
# 注意：部分非 OpenAI 上游对 type=openai 的请求会返回 400，
# 因此推荐默认使用 type=anthropic。仅在确认所选模型走 OpenAI 兼容上游时再用此方式。
export COPILOT_PROVIDER_TYPE="openai"
export COPILOT_PROVIDER_BASE_URL="https://your-domain.com/v1"
export COPILOT_PROVIDER_API_KEY="YOUR_API_KEY"
export COPILOT_MODEL="gpt-5.4-codex"`;

  const copilotRunCode = `# 启动 Copilot CLI（按账户偏好可能是 \`copilot\` 或 \`gh copilot\`）
copilot

# 直接提问
copilot "解释这个仓库的目录结构"`;

  const copilotModelsCode = `# 与其他工具一致，可使用网关中所有聊天模型

# Claude 系列（推荐，配合 type=anthropic 体验最完整）
claude-sonnet-4-6           # Sonnet 4.6
claude-opus-4-6             # Opus 4.6
claude-haiku-4-5-20251001   # Haiku 4.5

# OpenAI 系列
gpt-5.4
gpt-5.4-codex
gpt-5.4-codex-high

# Gemini 系列
gemini-3.1-pro-preview      # Gemini 3.1 Pro Preview

# 智谱 GLM 系列
glm-5.1`;

  // --- Trae examples ---

  const traeUiCode = `// Trae UI 配置步骤
// 步骤 1：打开 Trae 侧边栏聊天框右上角齿轮 → Settings → Models
// 步骤 2：点击 + Add Model
// 步骤 3：Provider 选择 OpenAI 或 OpenAI Compatible
// 步骤 4：填入下列字段并保存
//   - API Key:  YOUR_API_KEY（控制台 API 密钥页创建）
//   - Base URL: https://your-domain.com/v1
//   - Model:    claude-sonnet-4-6（或网关中其他聊天模型）
// 步骤 5：保存后回到聊天框右上角，重新选择刚添加的模型`;

  const traeModelsCode = `# Trae 可用网关中所有聊天模型
claude-sonnet-4-6
claude-opus-4-6
gpt-5.4
gpt-5.4-codex
gemini-3.1-pro-preview
glm-5.1`;

  // --- OpenCode examples ---

  const opencodeAddProviderCode = `// OpenCode TUI 添加 Provider
// 步骤 1：启动 opencode（交互模式）
// 步骤 2：按 / 或对应快捷键打开命令面板，进入 Settings
//        （部分版本叫 Providers / Models / Custom Endpoints）
// 步骤 3：选择 Add Provider → 选 OpenAI Compatible
// 步骤 4：依次填入下列字段并保存
//   - Name:    LLM Gateway（自定义）
//   - Base URL: https://your-domain.com/v1
//   - API Key:  YOUR_API_KEY
//   - Model:    claude-sonnet-4-6（或其他网关模型名）
// 步骤 5：保存后 OpenCode 自动测试连接，重启 opencode 后
//        在模型选择面板中即可看到 LLM Gateway 分组`;

  const opencodeRunCode = `# 启动 OpenCode，选择网关模型即可使用
opencode

# 直接执行任务
opencode run "解释这个仓库结构"`;

  const errorsContent = [
    { code: "400", title: "Bad Request", desc: "请求格式错误或缺少必填参数" },
    { code: "401", title: "Unauthorized", desc: "API Key 无效或未提供" },
    { code: "402", title: "Payment Required", desc: "账户余额不足" },
    { code: "403", title: "Forbidden", desc: "账户已被禁用或模型未配置定价" },
    { code: "404", title: "Not Found", desc: "请求的模型不存在或未配置" },
    { code: "405", title: "Method Not Allowed", desc: "HTTP 方法不被支持（如对 POST 端点发送 GET）" },
    { code: "413", title: "Request Entity Too Large", desc: "请求体超过大小限制（默认 20MB）" },
    { code: "429", title: "Too Many Requests", desc: "请求频率超过限制" },
    { code: "500", title: "Internal Server Error", desc: "服务器内部错误，请重试" },
  ];

  const errorCodes = [
    { code: "bad_json", desc: "请求体 JSON 格式错误", retryable: false },
    { code: "missing_fields", desc: "缺少必填字段", retryable: false },
    { code: "missing_model", desc: "未指定模型名称", retryable: false },
    { code: "invalid_api_key", desc: "API Key 无效或已吊销", retryable: false },
    { code: "no_user", desc: "用户不存在", retryable: false },
    { code: "insufficient_balance", desc: "账户余额不足", retryable: false },
    { code: "no_pricing", desc: "模型未配置定价", retryable: false },
    { code: "quota_exceeded", desc: "托管 Key 配额已用完", retryable: false },
    { code: "not_found", desc: "请求的资源不存在", retryable: false },
    { code: "request_too_large", desc: "请求体超过大小限制", retryable: false },
    { code: "too_frequent", desc: "请求频率超限，请稍后重试", retryable: true },
    { code: "db_error", desc: "数据库操作失败", retryable: true },
    { code: "upstream_read_error", desc: "上游服务响应异常", retryable: true },
    { code: "sms_error", desc: "短信发送失败", retryable: true },
  ];

  return (
    <div className="w-full h-screen flex flex-col">
      <Seo path="/docs" />
      {/* Top bar */}
      <div className="h-14 bg-card border-b border-border flex items-center justify-between px-6 flex-shrink-0 shadow-card">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 brand-gradient rounded-lg flex items-center justify-center">
              <Zap size={14} className="text-primary-foreground" />
            </div>
            <span className="font-bold text-sm text-foreground">LLM Gateway</span>
          </div>
          <div className="flex items-center gap-1.5 text-muted-foreground text-sm">
            <ChevronRight size={14} />
            <div className="flex items-center gap-1.5">
              <BookOpen size={14} className="text-primary" />
              <span className="text-foreground font-medium">API 文档</span>
            </div>
          </div>
        </div>
        <button
          onClick={() => navigate(auth.isAuthenticated ? "/dashboard" : "/")}
          className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          <ArrowLeft size={15} />
          {auth.isAuthenticated ? "返回控制台" : "返回首页"}
        </button>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <div className="w-[200px] flex-shrink-0 border-r border-border bg-sidebar flex flex-col py-5">
          <div className="px-4 mb-3">
            <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">导航</div>
          </div>
          <div className="px-2">
            {menuItems.map((item) => (
              <button
                key={item.key}
                onClick={() => {
                setActiveSection(item.key);
                document.getElementById(`section-${item.key}`)?.scrollIntoView({ behavior: "smooth", block: "start" });
              }}
                className={`w-full text-left px-3 py-2.5 rounded-lg text-sm transition-all duration-200 mb-0.5 ${
                  activeSection === item.key
                    ? "bg-primary/8 text-primary font-semibold border-l-2 border-primary"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted"
                }`}
              >
                {item.label}
              </button>
            ))}
            <a
              href="/api-reference.html"
              target="_blank"
              rel="noopener noreferrer"
              className="block w-full text-left px-3 py-2.5 rounded-lg text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-all duration-200 mb-0.5"
            >
              API 参考（OpenAPI）↗
            </a>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto bg-background">
          <div className="max-w-3xl mx-auto px-10 py-8">
            {/* Hero banner */}
            <div className="balance-card-gradient rounded-2xl px-8 py-7 mb-8 relative overflow-hidden">
              <div className="absolute top-0 right-0 w-48 h-48 rounded-full pointer-events-none"
                style={{ background: "radial-gradient(circle, rgba(37,99,235,0.3) 0%, transparent 70%)", transform: "translate(30%, -30%)" }} />
              <div className="relative z-10">
                <h1 className="text-2xl font-bold text-primary-foreground mb-2">API 接口文档</h1>
                <p className="text-sm leading-relaxed mb-4" style={{ color: "rgba(148,163,184,1)" }}>
                  LLM Gateway 提供兼容 OpenAI、Anthropic 和 Google Gemini 格式的 API 接口,支持所有主流 AI 编码工具无缝接入。
                </p>
                <a
                  href="https://www.bilibili.com/video/BV1afLV6NExm/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 px-3.5 py-2 rounded-lg bg-white/10 hover:bg-white/15 border border-white/15 transition-colors text-sm text-primary-foreground"
                >
                  <PlayCircle size={16} className="text-primary-foreground" />
                  <span>参考视频:平台配置 vibe coding 工具教程(B 站)</span>
                </a>
              </div>
            </div>

            {/* API Overview section */}
            <section id="section-api-overview" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">API 概览</h2>
              <p className="text-sm text-muted-foreground mb-4">
                Gateway 提供四种 LLM API 格式，兼容不同工具和 SDK 的调用需求。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">API 端点</h3>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">端点</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">格式</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">适用工具</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /v1/chat/completions</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">OpenAI Chat Completions</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Cursor, Trae, OpenCode, Windsurf, Aider</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /v1/messages</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Anthropic Messages</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Claude Code, Copilot CLI（BYOK）, Aider</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /v1/responses</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">OpenAI Responses</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Codex CLI</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /v1/images/tasks</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">异步图像任务（生成 / 编辑 / 查询 / 删除）</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">SDK / REST</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">GET /v1/models</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">OpenAI 兼容</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">所有工具</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">GET /v1/balance</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">查询余额 / 配额</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">SDK / REST</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /v1/messages/count_tokens</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Anthropic Token 计数</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">SDK / REST</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /gemini/v1/models/*</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Google Gemini 原生</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Gemini SDK, REST</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /gemini/v1beta/models/*</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Google Gemini 原生（含图像生成）</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Gemini SDK, REST</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">认证方式</h3>
              <p className="text-sm text-muted-foreground mb-5">
                所有端点支持 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">Authorization: Bearer</code> 认证，Messages API 额外支持 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">x-api-key</code>。详见<button onClick={() => { setActiveSection("auth"); document.getElementById("section-auth")?.scrollIntoView({ behavior: "smooth" }); }} className="text-primary hover:underline mx-1">认证方式</button>章节。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">Base URL 规则</h3>
              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p><strong>OpenAI 格式</strong>（Chat Completions / Responses / Models）— Base URL 需要以 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">/v1</code> 结尾：<code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">https://your-domain.com/v1</code></p>
                  <p className="mt-1"><strong>Anthropic 格式</strong>（Messages API）— Base URL 不要以 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">/v1</code> 结尾：<code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">https://your-domain.com</code></p>
                  <p className="mt-1"><strong>Gemini 格式</strong> — SDK 使用 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">https://your-domain.com/gemini</code>（SDK 自动拼接版本路径）；REST 调用使用 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">https://your-domain.com/gemini/v1</code> 或 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">https://your-domain.com/gemini/v1beta</code></p>
                </div>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">模型名写法</h3>
              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p>网关已做兼容：display name（如 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">claude-sonnet-4-6</code>）和带 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">gw/</code> 前缀（如 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">gw/claude-sonnet-4-6</code>）两种形式都支持，效果等价，任选其一即可。</p>
                  <p className="mt-1"><strong>例外</strong>：在 Cursor 中<strong>必须</strong>使用带 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">gw/</code> 前缀的写法，否则 Cursor 会按官方 OpenAI 模型名拦截/改写请求。</p>
                </div>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Quickstart section */}
            <section id="section-quickstart" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">快速开始</h2>
              <p className="text-sm text-muted-foreground mb-4">
                三步接入 LLM Gateway，开始使用所有已配置的 AI 模型。
              </p>

              <div className="space-y-4 mb-5">
                <div className="flex gap-3">
                  <div className="w-6 h-6 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-bold flex-shrink-0 mt-0.5">1</div>
                  <div className="flex-1">
                    <div className="text-sm font-semibold text-foreground mb-1">注册账号</div>
                    <p className="text-sm text-muted-foreground mb-3">在控制台注册账号。新用户自动获得试用额度，可直接体验。</p>
                    <DocImage
                      src={`${TOS_IMG_BASE}/%E6%8E%A7%E5%88%B6%E5%8F%B0%E7%99%BB%E5%BD%95%E9%A1%B5.png`}
                      alt="控制台登录页"
                      caption="控制台登录页 — 使用手机号注册并登录"
                    />
                  </div>
                </div>
                <div className="flex gap-3">
                  <div className="w-6 h-6 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-bold flex-shrink-0 mt-0.5">2</div>
                  <div className="flex-1">
                    <div className="text-sm font-semibold text-foreground mb-1">创建 API Key</div>
                    <p className="text-sm text-muted-foreground mb-3">进入控制台「API 密钥」页面，点击右上角「创建密钥」。在弹窗中给 Key 起个名字（如 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">claude-code-laptop</code>），点击确认后会<strong>只展示一次</strong>完整 Key（形如 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">sk-xxx</code>），立即复制保存。</p>
                    <DocImage
                      src={`${TOS_IMG_BASE}/API%E5%AF%86%E9%92%A5.png`}
                      alt="API 密钥列表页"
                      caption="API 密钥列表页 — 点击右上角「创建密钥」"
                    />
                    <DocImage
                      src={`${TOS_IMG_BASE}/%E5%88%9B%E5%BB%BA%E5%AF%86%E9%92%A5.png`}
                      alt="创建密钥弹窗"
                      caption="创建密钥弹窗 — Key 仅展示一次，关闭后只能在列表里看到掩码"
                    />
                  </div>
                </div>
                <div className="flex gap-3">
                  <div className="w-6 h-6 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-bold flex-shrink-0 mt-0.5">3</div>
                  <div className="flex-1">
                    <div className="text-sm font-semibold text-foreground mb-1">配置工具</div>
                    <p className="text-sm text-muted-foreground">将 Base URL 和 API Key 填入你的 AI 工具（Claude Code、Cursor、Codex 等），即可开始使用。</p>
                  </div>
                </div>
              </div>

              <CodeBlock lang="BASH" code={`# 示例：用 curl 发送第一个请求
curl -X POST https://your-domain.com/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model": "claude-sonnet-4-6", "messages": [{"role": "user", "content": "Hello!"}]}'`} blockId="quickstart-curl" copiedId={copiedId} onCopy={handleCopy} />
            </section>

            <div className="border-t border-border mb-8" />

            {/* Auth section (REQ-PO-30) */}
            <section id="section-auth" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">认证方式</h2>
              <p className="text-sm text-muted-foreground mb-4">
                Gateway 支持两种认证方式，所有端点均可使用 Bearer Token。Messages API 额外支持 Anthropic 风格的 x-api-key。
              </p>

              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">方式</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">Header</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">适用端点</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">状态</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Bearer Token</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">Authorization: Bearer sk-xxx</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">所有端点</TableCell>
                      <TableCell className="px-4 py-2.5"><span className="px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-700">推荐</span></TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">x-api-key</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">x-api-key: sk-xxx</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">仅 /v1/messages</TableCell>
                      <TableCell className="px-4 py-2.5"><span className="px-2 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-700">兼容</span></TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p>两种方式使用同一个 API Key（在控制台「API 密钥」页面创建）。Bearer Token 是通用方式，适用于所有端点；x-api-key 仅为兼容 Anthropic SDK 的默认行为而保留。新项目建议统一使用 Bearer Token。</p>
                </div>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">各工具的认证配置</h3>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">工具</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">认证方式</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">配置变量</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Claude Code</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">x-api-key（自动）</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">ANTHROPIC_AUTH_TOKEN</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Codex CLI</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Bearer Token（自动）</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">OPENAI_API_KEY</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Cursor</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Bearer Token（自动）</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">Settings → Models → API Key</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">REST / SDK</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Bearer Token</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">Authorization header</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Compatibility Matrix (REQ-PO-07) */}
            <section id="section-compatibility" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">多协议兼容性矩阵</h2>
              <p className="text-sm text-muted-foreground mb-4">
                Gateway 同时支持多种 API 协议，并在协议间自动转换。以下矩阵展示各端点支持的功能。
              </p>

              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">功能</TableHead>
                      <TableHead className="text-center px-3 py-2.5 font-medium text-muted-foreground">Chat Completions</TableHead>
                      <TableHead className="text-center px-3 py-2.5 font-medium text-muted-foreground">Messages API</TableHead>
                      <TableHead className="text-center px-3 py-2.5 font-medium text-muted-foreground">Responses API</TableHead>
                      <TableHead className="text-center px-3 py-2.5 font-medium text-muted-foreground">Gemini API</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {[
                      ["文本生成", true, true, true, true],
                      ["流式响应 (SSE)", true, true, true, true],
                      ["Tool Use / 函数调用", true, true, true, true],
                      ["多模态（图片输入）", true, true, false, true],
                      ["图像生成", false, false, false, "v1beta"],
                      ["Token 计数", false, true, false, false],
                      ["Extended Thinking", false, "Claude only", false, false],
                      ["Prompt Caching", false, "Claude only", false, false],
                      ["缓存内容 (cachedContents)", false, false, false, true],
                      ["跨模型路由", true, true, true, "Gemini only"],
                      ["自动格式转换", "—", "→ OpenAI", "→ Chat", "—"],
                    ].map(([feature, ...cols], i) => (
                      <TableRow key={i} className="border-b border-border last:border-0">
                        <TableCell className="px-4 py-2.5 text-foreground text-xs font-medium">{feature as string}</TableCell>
                        {(cols as (boolean | string)[]).map((v, j) => (
                          <TableCell key={j} className="text-center px-3 py-2.5 text-xs">
                            {v === true ? <span className="text-emerald-600">&#10003;</span> :
                             v === false ? <span className="text-muted-foreground">&#10005;</span> :
                             <span className="text-muted-foreground">{v}</span>}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">格式转换说明</h3>
              <div className="space-y-2 mb-5">
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>Messages API 可路由到 OpenAI 上游 — Gateway 自动在 Anthropic 和 OpenAI 格式间转换</span>
                </div>
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>Responses API 请求在内部转换为 Chat Completions 格式发送到上游，响应再转换回 Responses 格式</span>
                </div>
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>Chat Completions 端点也接受 Responses API 格式的请求体，自动识别并转换</span>
                </div>
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>Gemini API 为原生透传，不做格式转换，仅支持 Gemini 系列模型</span>
                </div>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">请求格式对照（OpenAI vs Anthropic）</h3>
              <p className="text-sm text-muted-foreground mb-3">
                同一个模型可通过不同协议调用，Gateway 自动转换。以下对照两种主要格式的字段映射：
              </p>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">概念</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">OpenAI (Chat Completions)</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">Anthropic (Messages)</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {[
                      ["认证", "Authorization: Bearer", "x-api-key / Authorization: Bearer"],
                      ["模型字段", "model", "model"],
                      ["消息列表", "messages[]", "messages[]"],
                      ["系统提示", 'messages[0].role="system"', "system (顶层字段)"],
                      ["最大输出", "max_tokens (可选)", "max_tokens (必填)"],
                      ["流式", "stream: true", "stream: true"],
                      ["工具调用", "tools[] + tool_choice", "tools[] + tool_choice"],
                      ["停止原因", 'finish_reason: "stop"', 'stop_reason: "end_turn"'],
                      ["用量统计", "usage.prompt_tokens / completion_tokens", "usage.input_tokens / output_tokens"],
                    ].map(([concept, openai, anthropic], i) => (
                      <TableRow key={i} className="border-b border-border last:border-0">
                        <TableCell className="px-4 py-2 text-foreground text-xs font-medium">{concept}</TableCell>
                        <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">{openai}</TableCell>
                        <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">{anthropic}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p>使用 Gateway 时无需关心格式差异 — 选择你的工具原生支持的端点即可。例如 Claude Code 使用 Messages API，Cursor 使用 Chat Completions，Gateway 在后端自动处理格式转换和路由。</p>
                </div>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Chat Completions section */}
            <section id="section-chat-completions" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Chat Completions API</h2>
              <div className="flex items-center gap-2 mb-3">
                <span className="px-2 py-0.5 rounded text-xs font-bold bg-green-100 text-green-700">POST</span>
                <code className="text-sm font-mono text-foreground">/v1/chat/completions</code>
              </div>
              <p className="text-sm text-muted-foreground mb-4">
                兼容 OpenAI Chat Completions 格式。支持所有已配置的模型（OpenAI、Claude、Gemini、Grok 等），Gateway 会自动路由到对应的上游。同时兼容 Responses API 格式的请求体，会自动转换。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">请求示例</h3>
              <CodeBlock lang="BASH" code={chatCompletionsCurl} blockId="chat-curl" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">响应示例</h3>
              <CodeBlock lang="JSON" code={chatCompletionsResponse} blockId="chat-resp" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">流式请求</h3>
              <p className="text-sm text-muted-foreground mb-3">
                设置 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">stream: true</code> 即可获得 SSE 流式响应：
              </p>
              <CodeBlock lang="BASH" code={chatCompletionsStreamCurl} blockId="chat-stream-curl" copiedId={copiedId} onCopy={handleCopy} />
            </section>

            <div className="border-t border-border mb-8" />

            {/* Messages API section */}
            <section id="section-messages-api" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Messages API</h2>
              <div className="flex items-center gap-2 mb-3">
                <span className="px-2 py-0.5 rounded text-xs font-bold bg-green-100 text-green-700">POST</span>
                <code className="text-sm font-mono text-foreground">/v1/messages</code>
              </div>
              <p className="text-sm text-muted-foreground mb-4">
                兼容 Anthropic Messages API 格式。支持 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">x-api-key</code> 和 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">Authorization: Bearer</code> 两种认证方式。支持 extended thinking、tool use、流式响应等特性。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">请求示例</h3>
              <CodeBlock lang="BASH" code={messagesApiCurl} blockId="messages-curl" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">响应示例</h3>
              <CodeBlock lang="JSON" code={messagesApiResponse} blockId="messages-resp" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p>Messages API 也支持将请求路由到 OpenAI 上游 — Gateway 会自动在 Anthropic 和 OpenAI 格式之间转换。</p>
                </div>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Responses API section */}
            <section id="section-responses-api" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Responses API</h2>
              <div className="flex items-center gap-2 mb-3">
                <span className="px-2 py-0.5 rounded text-xs font-bold bg-green-100 text-green-700">POST</span>
                <code className="text-sm font-mono text-foreground">/v1/responses</code>
              </div>
              <p className="text-sm text-muted-foreground mb-4">
                兼容 OpenAI Responses API 格式。Gateway 内部将请求转换为 Chat Completions 格式发送到上游，再将响应转换回 Responses 格式返回。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">请求示例</h3>
              <CodeBlock lang="BASH" code={responsesApiCurl} blockId="responses-curl" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">响应示例</h3>
              <CodeBlock lang="JSON" code={responsesApiResponse} blockId="responses-resp" copiedId={copiedId} onCopy={handleCopy} />
            </section>

            <div className="border-t border-border mb-8" />

            {/* Image Tasks (async) section */}
            <section id="section-image-tasks" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">图像任务（异步）</h2>
              <p className="text-sm text-muted-foreground mb-4">
                图像生成 / 编辑（如 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">gpt-image-2</code>、<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">gemini-3-pro-image-preview</code>）耗时较长，直接调用同步接口容易触发 HTTP 超时。异步任务接口让你<strong>提交任务后立即拿到任务 ID</strong>，再通过轮询查询结果，单个任务可返回多张图片。支持普通用户 Key、租户 Key、租户子用户 Key 三类 API Key，出图后按对应账户计费。
              </p>

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p><strong>使用流程</strong>：提交任务（<code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">202</code> 返回 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">id</code>）→ 轮询任务直到 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">status</code> 变为 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">completed</code> → 从 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">result_urls</code> 取图片直链。</p>
                  <p className="mt-1"><strong>状态取值</strong>：<code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">pending</code>（排队）/ <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">processing</code>（处理中）/ <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">completed</code>（完成）/ <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">failed</code>（失败，见 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">error_message</code>）。</p>
                </div>
              </div>

              <div className="flex items-center gap-2 mb-3">
                <span className="px-2 py-0.5 rounded text-xs font-bold bg-green-100 text-green-700">POST</span>
                <code className="text-sm font-mono text-foreground">/v1/images/tasks</code>
                <span className="text-sm text-muted-foreground">— 提交生成任务</span>
              </div>
              <p className="text-sm text-muted-foreground mb-3">
                请求体字段：<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">model</code>（必填）、<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">prompt</code>（必填，≤ 2000 字）、<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">size</code>（默认 1024x1024）、<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">n</code>（出图张数，1-4，默认 1）、<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">params</code>（可选，透传上游参数）。
              </p>
              <CodeBlock lang="BASH" code={imageTaskSubmitCurl} blockId="imgtask-submit" copiedId={copiedId} onCopy={handleCopy} />
              <h3 className="text-sm font-semibold text-foreground mb-2">响应示例（202 Accepted）</h3>
              <CodeBlock lang="JSON" code={imageTaskSubmitResponse} blockId="imgtask-submit-resp" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex items-center gap-2 mb-3 mt-6">
                <span className="px-2 py-0.5 rounded text-xs font-bold bg-green-100 text-green-700">POST</span>
                <code className="text-sm font-mono text-foreground">/v1/images/tasks/edits</code>
                <span className="text-sm text-muted-foreground">— 提交编辑任务</span>
              </div>
              <p className="text-sm text-muted-foreground mb-3">
                在生成字段基础上，通过 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">image_urls</code>（图片链接数组）和 / 或 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">image_base64s</code>（base64 数组）提供输入图，二者可任选其一或混用；可选 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">mask_base64</code> 提供蒙版。输入图最多 4 张、单张 ≤ 25MB。
              </p>
              <CodeBlock lang="BASH" code={imageTaskEditCurl} blockId="imgtask-edit" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex items-center gap-2 mb-3 mt-6">
                <span className="px-2 py-0.5 rounded text-xs font-bold bg-blue-100 text-blue-700">GET</span>
                <code className="text-sm font-mono text-foreground">/v1/images/tasks/{"{id}"}</code>
                <span className="text-sm text-muted-foreground">— 查询任务状态 / 结果</span>
              </div>
              <CodeBlock lang="BASH" code={imageTaskPollCurl} blockId="imgtask-poll" copiedId={copiedId} onCopy={handleCopy} />
              <h3 className="text-sm font-semibold text-foreground mb-2">响应示例（200，完成时）</h3>
              <CodeBlock lang="JSON" code={imageTaskPollResponse} blockId="imgtask-poll-resp" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex items-center gap-2 mb-3 mt-6">
                <span className="px-2 py-0.5 rounded text-xs font-bold bg-red-100 text-red-700">DELETE</span>
                <code className="text-sm font-mono text-foreground">/v1/images/tasks/{"{id}"}</code>
                <span className="text-sm text-muted-foreground">— 删除任务</span>
              </div>
              <p className="text-sm text-muted-foreground mb-3">
                用于删除自己的任务（例如长时间排队未处理的任务）。仅可删除 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">pending</code> / <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">completed</code> / <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">failed</code> 状态；<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">processing</code>（处理中）的任务会返回 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">409</code>，请稍后重试。成功返回 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">204 No Content</code>。
              </p>
              <CodeBlock lang="BASH" code={imageTaskDeleteCurl} blockId="imgtask-delete" copiedId={copiedId} onCopy={handleCopy} />
            </section>

            <div className="border-t border-border mb-8" />

            {/* Models section */}
            <section id="section-models" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">模型列表</h2>
              <div className="flex items-center gap-2 mb-3">
                <span className="px-2 py-0.5 rounded text-xs font-bold bg-blue-100 text-blue-700">GET</span>
                <code className="text-sm font-mono text-foreground">/v1/models</code>
              </div>
              <p className="text-sm text-muted-foreground mb-4">
                返回当前可用的模型列表（OpenAI 兼容格式）。需要 API Key 认证。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">请求示例</h3>
              <CodeBlock lang="BASH" code={modelsApiCurl} blockId="models-curl" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">响应示例</h3>
              <CodeBlock lang="JSON" code={modelsApiResponse} blockId="models-resp" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-3">完整模型列表</h3>

              <h4 className="text-sm font-semibold text-foreground mb-2">Anthropic Claude 系列</h4>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-4">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">模型名称</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">类型</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {[
                      ["claude-sonnet-4-6", "chat"],
                      ["claude-opus-4-6", "chat"],
                      ["claude-haiku-4-5-20251001", "chat"],
                    ].map(([name, type], i, arr) => (
                      <TableRow key={name} className={i < arr.length - 1 ? "border-b border-border" : ""}>
                        <TableCell className="px-4 py-2 font-mono text-xs">{name}</TableCell>
                        <TableCell className="px-4 py-2 text-muted-foreground">{type}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <h4 className="text-sm font-semibold text-foreground mb-2">OpenAI 系列</h4>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-4">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">模型名称</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">类型</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {[
                      ["gpt-5.4", "chat"],
                      ["gpt-5.4-codex", "chat"],
                      ["gpt-5.4-codex-high", "chat"],
                    ].map(([name, type], i, arr) => (
                      <TableRow key={name} className={i < arr.length - 1 ? "border-b border-border" : ""}>
                        <TableCell className="px-4 py-2 font-mono text-xs">{name}</TableCell>
                        <TableCell className="px-4 py-2 text-muted-foreground">{type}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <h4 className="text-sm font-semibold text-foreground mb-2">Google Gemini 系列</h4>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-4">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">模型名称</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">类型</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {[
                      ["gemini-3.1-pro-preview", "chat"],
                    ].map(([name, type], i, arr) => (
                      <TableRow key={name} className={i < arr.length - 1 ? "border-b border-border" : ""}>
                        <TableCell className="px-4 py-2 font-mono text-xs">{name}</TableCell>
                        <TableCell className="px-4 py-2 text-muted-foreground">{type}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <h4 className="text-sm font-semibold text-foreground mb-2">智谱 GLM 系列</h4>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-4">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">模型名称</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">类型</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {[
                      ["glm-5.1", "chat"],
                    ].map(([name, type], i, arr) => (
                      <TableRow key={name} className={i < arr.length - 1 ? "border-b border-border" : ""}>
                        <TableCell className="px-4 py-2 font-mono text-xs">{name}</TableCell>
                        <TableCell className="px-4 py-2 text-muted-foreground">{type}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p>以上为常用模型，完整列表请调用 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">GET /v1/models</code> 获取。模型列表会随网关配置动态更新。</p>
                </div>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Balance section */}
            <section id="section-balance" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">查询余额</h2>
              <div className="flex items-center gap-2 mb-3">
                <span className="px-2 py-0.5 rounded text-xs font-bold bg-blue-100 text-blue-700">GET</span>
                <code className="text-sm font-mono text-foreground">/v1/balance</code>
              </div>
              <p className="text-sm text-muted-foreground mb-4">
                查询当前 API Key 对应账户的余额或配额。需要 API Key 认证。响应中的 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">type</code> 字段标识账户类型，不同类型返回字段不同。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">请求示例</h3>
              <CodeBlock lang="BASH" code={balanceApiCurl} blockId="balance-curl" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">响应示例（普通用户 Key）</h3>
              <CodeBlock lang="JSON" code={balanceApiResponse} blockId="balance-resp" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">不同 Key 类型返回字段</h3>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-4">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">type</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">字段</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2 font-mono text-xs">user</TableCell>
                      <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">balance, frozen, available, currency</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2 font-mono text-xs">tenant</TableCell>
                      <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">balance, frozen, available, total_recharged, total_consumed, currency</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2 font-mono text-xs">sub_user</TableCell>
                      <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">quota_limit（null 为无限制）, quota_used, quota_remaining, currency</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </section>

            <div className="border-t border-border mb-8" />
            <section id="section-claude-code" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Claude Code 配置</h2>
              <p className="text-sm text-muted-foreground mb-4">
                <a href="https://docs.anthropic.com/en/docs/claude-code" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">Claude Code</a> 是
                Anthropic 推出的 AI 编程助手 CLI，原生使用 Anthropic Messages 协议。通过本网关，不仅能用 Claude，还能用 GPT、Gemini、GLM —— 网关会自动转协议。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">1. 安装 Claude Code</h3>
              <p className="text-sm text-muted-foreground mb-3">
                需要 Node.js &ge; 18。使用 npm 全局安装：
              </p>
              <CodeBlock lang="BASH" code="npm install -g @anthropic-ai/claude-code" blockId="claude-install" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">2. 配置环境变量</h3>
              <p className="text-sm text-muted-foreground mb-3">
                将 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">ANTHROPIC_BASE_URL</code> 指向本网关地址，并设置 API Key、默认模型与三个档位映射变量：
              </p>
              <CodeBlock lang="BASH" code={claudeCodeEnvCode} blockId="claude-env" copiedId={copiedId} onCopy={handleCopy} />

              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">变量</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">说明</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">必填</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">ANTHROPIC_BASE_URL</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">https://your-domain.com</code>，<strong>不</strong>带 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">/v1</code></TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">✅</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">ANTHROPIC_AUTH_TOKEN</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">网关 API Key（控制台 → API 密钥页创建）</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">✅</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">ANTHROPIC_MODEL</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">默认模型，可填网关任意聊天模型</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">可选</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">ANTHROPIC_DEFAULT_SONNET_MODEL</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Sonnet 档位映射，建议 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">claude-sonnet-4-6</code></TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">可选</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">ANTHROPIC_DEFAULT_OPUS_MODEL</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Opus 档位映射，建议 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">claude-opus-4-6</code></TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">可选</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">ANTHROPIC_DEFAULT_HAIKU_MODEL</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Haiku 档位映射，建议 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">claude-haiku-4-5-20251001</code></TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">可选</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p><strong>关于三个档位变量</strong> — Claude Code 在 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">/model</code> 菜单中把模型分成 Sonnet / Opus / Haiku 三档，这三个变量决定每档实际请求哪个网关模型。建议显式设置，避免 Claude Code 内置默认值与网关模型名不一致。</p>
                </div>
              </div>

              <DocImage
                src={`${TOS_IMG_BASE}/claudecode%E7%8E%AF%E5%A2%83%E9%85%8D%E7%BD%AE.png`}
                alt="终端写入 ~/.zshrc 演示"
                caption="将上述环境变量写入 ~/.zshrc（bash 用户写入 ~/.bashrc），然后 source 让其生效"
              />

              <h3 className="text-sm font-semibold text-foreground mb-2">3. 可用模型</h3>
              <CodeBlock lang="BASH" code={claudeCodeModelsCode} blockId="claude-models" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p><strong>格式自动转换</strong> — 对 Claude 模型，请求直接透传到 Anthropic 上游；对非 Claude 模型（GPT、Gemini、GLM 等），网关自动在 Anthropic Messages 和 OpenAI Chat Completions 格式之间转换。</p>
                  <p className="mt-1"><strong>功能差异</strong> — Extended Thinking 和 Prompt Caching 仅在原生 Claude 模型上可用。Tool Use（函数调用）跨所有模型均可用。</p>
                  <p className="mt-1"><strong>模型名写法</strong> — display name（如 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">claude-sonnet-4-6</code>）和带 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">gw/</code> 前缀（如 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">gw/claude-sonnet-4-6</code>）两种形式都支持，效果等价。</p>
                </div>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">4. 启动使用</h3>
              <CodeBlock lang="BASH" code={claudeCodeRunCode} blockId="claude-run" copiedId={copiedId} onCopy={handleCopy} />

              <p className="text-sm text-muted-foreground mb-3">
                进入交互界面后，输入 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">/model</code> 即可切换网关支持的任意模型：
              </p>
              <DocImage
                src={`${TOS_IMG_BASE}/claude%E9%80%89%E6%8B%A9%E6%A8%A1%E5%9E%8B.png`}
                alt="Claude Code /model 切换面板"
                caption="Claude Code /model 切换面板 — 三档分别对应上一步配置的 Sonnet / Opus / Haiku 网关模型"
              />

              <h3 className="text-sm font-semibold text-foreground mb-2">5. 持久化配置</h3>
              <p className="text-sm text-muted-foreground mb-3">
                将环境变量写入 shell 配置文件，避免每次手动设置：
              </p>
              <CodeBlock lang="BASH" code={claudeCodeSettingsCode} blockId="claude-settings" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2 mt-6">常见问题</h3>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">现象</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">原因 / 解决</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">启动报「Connection refused」</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">检查 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">ANTHROPIC_BASE_URL</code>，<strong>不要</strong>在末尾加 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">/v1</code></TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">401 Unauthorized</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">ANTHROPIC_AUTH_TOKEN</code> 是否填写网关 Key（<code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">sk-</code> 开头）</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">报「model not found」</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">检查模型名称是否输入正确，与网关「模型管理」页保持一致即可（带不带 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">gw/</code> 前缀都行）</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 text-foreground">切换模型不生效</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">/model</code> 命令是会话级；想永久切换请改 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">ANTHROPIC_MODEL</code></TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Claude Tips section */}
            <section id="section-claude-tips" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Claude 省钱技巧</h2>
              <p className="text-sm text-muted-foreground mb-4">
                使用 Claude 模型时，通过合理的策略可以显著降低成本。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">1. 善用 Prompt Cache</h3>
              <p className="text-sm text-muted-foreground mb-3">
                Prompt Cache 可以缓存重复的上下文，大幅降低输入 token 成本：
              </p>
              <ul className="text-sm text-muted-foreground mb-4 space-y-2 list-disc list-inside">
                <li><strong>5 分钟 TTL</strong>：适合连续对话场景，缓存读取价格是正常输入的 10%</li>
                <li><strong>1 小时 TTL</strong>：适合长时间编码会话，缓存读取价格是正常输入的 10%</li>
                <li><strong>最佳实践</strong>：将项目上下文、代码库结构等固定内容放在消息开头，标记为可缓存</li>
              </ul>

              <h3 className="text-sm font-semibold text-foreground mb-2">2. 按任务选择模型</h3>
              <p className="text-sm text-muted-foreground mb-3">
                不同模型适合不同场景，合理搭配可以节省 50% 以上成本：
              </p>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">场景</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">推荐模型</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">原因</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">简单代码补全、注释生成</TableCell>
                      <TableCell className="px-4 py-2.5 text-foreground font-medium">Haiku 4.5</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">速度快、成本低</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">日常编码、代码审查</TableCell>
                      <TableCell className="px-4 py-2.5 text-foreground font-medium">Sonnet 4.6</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">性价比最高</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 text-foreground">架构设计、复杂重构</TableCell>
                      <TableCell className="px-4 py-2.5 text-foreground font-medium">Opus 4.6</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">推理能力最强</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">3. 订阅套餐 vs 按量付费</h3>
              <p className="text-sm text-muted-foreground mb-3">
                根据使用频率选择合适的付费方式：
              </p>
              <ul className="text-sm text-muted-foreground mb-4 space-y-2 list-disc list-inside">
                <li><strong>每天使用 &lt; 2 小时</strong>：按量付费更划算</li>
                <li><strong>每天使用 2-6 小时</strong>：Pro 或 Plus 套餐（¥99-299/月）</li>
                <li><strong>每天使用 &gt; 6 小时</strong>：Premium 或 Max 套餐（¥599-999/月）</li>
              </ul>

              <h3 className="text-sm font-semibold text-foreground mb-2">4. 控制上下文长度</h3>
              <p className="text-sm text-muted-foreground mb-3">
                避免发送不必要的上下文：
              </p>
              <ul className="text-sm text-muted-foreground mb-4 space-y-2 list-disc list-inside">
                <li>定期清理对话历史，只保留最近 10-20 轮</li>
                <li>使用 Claude Code 的 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">--context-window</code> 参数限制上下文</li>
                <li>避免重复发送大文件内容，使用文件引用</li>
              </ul>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Claude Models Guide section */}
            <section id="section-claude-models" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Claude 模型选择指南</h2>
              <p className="text-sm text-muted-foreground mb-4">
                Claude 提供三个系列的模型，各有特点和适用场景。
              </p>

              <div className="space-y-5">
                <div className="bg-card border border-border rounded-xl p-5">
                  <div className="flex items-center gap-2 mb-3">
                    <h3 className="text-base font-bold text-foreground">Claude Haiku 4.5</h3>
                    <span className="px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-700 border border-green-200">
                      最快速
                    </span>
                  </div>
                  <p className="text-sm text-muted-foreground mb-3">
                    轻量级模型，响应速度最快，适合高频次的简单任务。
                  </p>
                  <div className="text-sm space-y-2">
                    <div><strong className="text-foreground">适用场景：</strong><span className="text-muted-foreground">代码补全、语法检查、简单问答、注释生成</span></div>
                    <div><strong className="text-foreground">价格：</strong><span className="text-muted-foreground">¥0.80/M 输入，¥4.00/M 输出</span></div>
                    <div><strong className="text-foreground">特点：</strong><span className="text-muted-foreground">速度快、成本低、适合批量处理</span></div>
                  </div>
                </div>

                <div className="bg-card border border-orange-400 ring-2 ring-orange-400/20 rounded-xl p-5">
                  <div className="flex items-center gap-2 mb-3">
                    <h3 className="text-base font-bold text-foreground">Claude Sonnet 4.6</h3>
                    <span className="px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700 border border-blue-200">
                      推荐
                    </span>
                  </div>
                  <p className="text-sm text-muted-foreground mb-3">
                    平衡性能与成本的中端模型，是大多数开发者的首选。
                  </p>
                  <div className="text-sm space-y-2">
                    <div><strong className="text-foreground">适用场景：</strong><span className="text-muted-foreground">日常编码、代码审查、Bug 修复、单元测试编写</span></div>
                    <div><strong className="text-foreground">价格：</strong><span className="text-muted-foreground">¥2.40/M 输入，¥12.00/M 输出（比官方便宜 20%）</span></div>
                    <div><strong className="text-foreground">特点：</strong><span className="text-muted-foreground">性价比最高、支持 Extended Thinking、支持 Prompt Cache</span></div>
                  </div>
                </div>

                <div className="bg-card border border-border rounded-xl p-5">
                  <div className="flex items-center gap-2 mb-3">
                    <h3 className="text-base font-bold text-foreground">Claude Opus 4.6</h3>
                    <span className="px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-700 border border-purple-200">
                      最强大
                    </span>
                  </div>
                  <p className="text-sm text-muted-foreground mb-3">
                    旗舰模型，推理能力最强，适合复杂任务和架构设计。
                  </p>
                  <div className="text-sm space-y-2">
                    <div><strong className="text-foreground">适用场景：</strong><span className="text-muted-foreground">架构设计、复杂重构、算法优化、技术方案评审</span></div>
                    <div><strong className="text-foreground">价格：</strong><span className="text-muted-foreground">¥4.00/M 输入，¥20.00/M 输出（比官方便宜 20%）</span></div>
                    <div><strong className="text-foreground">特点：</strong><span className="text-muted-foreground">推理能力最强、支持 Extended Thinking、适合复杂问题</span></div>
                  </div>
                </div>
              </div>

              <div className="flex gap-2 bg-orange-50 border border-orange-200 rounded-xl p-4 mt-5">
                <div className="w-1 bg-orange-500 rounded-full flex-shrink-0" />
                <div className="text-sm text-orange-800">
                  <p><strong>💡 省钱建议</strong></p>
                  <p className="mt-1">日常编码用 Sonnet 4.6，简单任务用 Haiku 4.5，复杂架构设计才用 Opus 4.6。合理搭配可以节省 50% 以上成本。</p>
                </div>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* CC Switch section */}
            <section id="section-cc-switch" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">CC Switch 配置</h2>
              <p className="text-sm text-muted-foreground mb-4">
                <a href="https://github.com/farion1231/cc-switch" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">CC Switch</a> 是一个跨平台的
                AI 编程工具配置管理器，支持在多个 API Provider 之间一键切换。通过 CC Switch 可以快速将 Claude Code、Codex、Cursor 等工具连接到本网关，无需手动修改环境变量。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">1. 安装 CC Switch</h3>
              <CodeBlock lang="BASH" code={ccSwitchInstallCode} blockId="ccswitch-install" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">2. 添加本网关为 Provider</h3>
              <p className="text-sm text-muted-foreground mb-3">
                打开 CC Switch，点击添加 Provider，填写以下信息：
              </p>
              <CodeBlock lang="TEXT" code={ccSwitchConfigCode} blockId="ccswitch-config" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p><strong>Base URL</strong> — 填写网关地址，不要以 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">/v1</code> 结尾。</p>
                  <p className="mt-1"><strong>API Key</strong> — 在控制台 <strong>API 密钥</strong> 页面创建，格式为 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">sk-xxx</code>。</p>
                  <p className="mt-1"><strong>模型列表</strong> — CC Switch 会自动调用 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">GET /v1/models</code> 获取可用模型。</p>
                </div>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">3. 支持的工具</h3>
              <p className="text-sm text-muted-foreground mb-3">
                本网关同时兼容多种 API 格式，CC Switch 管理的所有主流工具均可使用：
              </p>
              <CodeBlock lang="TEXT" code={ccSwitchToolsTable} blockId="ccswitch-tools" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">4. 手动配置（不使用 CC Switch）</h3>
              <p className="text-sm text-muted-foreground mb-3">
                如果不想安装 CC Switch，也可以直接设置环境变量：
              </p>
              <CodeBlock lang="BASH" code={ccSwitchManualCode} blockId="ccswitch-manual" copiedId={copiedId} onCopy={handleCopy} />
            </section>

            <div className="border-t border-border mb-8" />

            {/* Codex CLI section */}
            <section id="section-codex-cli" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Codex CLI 配置</h2>
              <p className="text-sm text-muted-foreground mb-4">
                <a href="https://github.com/openai/codex" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">Codex CLI</a> 是
                OpenAI 推出的开源 AI 编程助手。默认走 Responses API，但接入网关时<strong>强烈建议改用 Chat Completions</strong>（<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">wire_api = "chat"</code>），兼容性最好。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">1. 安装 Codex CLI</h3>
              <p className="text-sm text-muted-foreground mb-3">
                需要 Node.js &ge; 22。使用 npm 全局安装：
              </p>
              <CodeBlock lang="BASH" code="npm install -g @openai/codex" blockId="codex-install" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">2. 三件套配置</h3>
              <p className="text-sm text-muted-foreground mb-3">
                Codex CLI 需要<strong>同时</strong>配齐三处：环境变量 + <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">~/.codex/config.toml</code> + <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">~/.codex/auth.json</code>，缺一启动报错。
              </p>

              <p className="text-sm text-muted-foreground mb-3 mt-4">
                <strong>步骤 1：环境变量</strong>
              </p>
              <CodeBlock lang="BASH" code={codexEnvCode} blockId="codex-env" copiedId={copiedId} onCopy={handleCopy} />

              <p className="text-sm text-muted-foreground mb-3 mt-4">
                <strong>步骤 2：<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">~/.codex/config.toml</code></strong>
              </p>
              <CodeBlock lang="TOML" code={codexConfigToml} blockId="codex-toml" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex gap-2 bg-amber-50 border border-amber-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-amber-400 rounded-full flex-shrink-0" />
                <div className="text-sm text-amber-800">
                  <p><strong>🔑 关键</strong>：<code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">wire_api = "chat"</code> 必须显式写。Codex 默认 <code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">responses</code>，会发 GET 预检请求导致兼容性问题。</p>
                </div>
              </div>

              <p className="text-sm text-muted-foreground mb-3 mt-4">
                <strong>步骤 3：<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">~/.codex/auth.json</code></strong>
              </p>
              <CodeBlock lang="JSON" code={codexAuthJson} blockId="codex-auth" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex gap-2 bg-amber-50 border border-amber-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-amber-400 rounded-full flex-shrink-0" />
                <div className="text-sm text-amber-800">
                  <p><strong>🔑 关键</strong>：<code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">OPENAI_API_KEY</code> 这个字段名 Codex 写死了，<strong>不能改成别的名字</strong>，否则会报 <code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">API key auth is missing a key</code>。</p>
                </div>
              </div>

              <p className="text-sm text-muted-foreground mb-3">
                如果 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">config.toml</code> 引用了其他自定义环境变量名，请同步设置：
              </p>
              <CodeBlock lang="BASH" code={codexConfigEnv} blockId="codex-config-env" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">3. 可用模型</h3>
              <CodeBlock lang="BASH" code={codexModelsCode} blockId="codex-models" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">4. 启动使用</h3>
              <CodeBlock lang="BASH" code={codexRunCode} blockId="codex-run" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">5. 持久化配置</h3>
              <p className="text-sm text-muted-foreground mb-3">
                将环境变量写入 shell 配置文件，避免每次手动设置：
              </p>
              <CodeBlock lang="BASH" code={codexPersistCode} blockId="codex-persist" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2 mt-6">常见问题</h3>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">现象</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">原因 / 解决</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">API key auth is missing a key</code></TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">~/.codex/auth.json</code> 里字段名必须是 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">OPENAI_API_KEY</code>，不能改名</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">启动后第一次请求 405 / 400</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">config.toml</code> 里 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">wire_api</code> 没设为 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">"chat"</code>，默认走 responses 不通</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">401 Unauthorized</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">三处的 Key 是否一致；环境变量是否 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">source</code> 生效</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 text-foreground">Base URL 报错</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Base URL 必须以 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">/v1</code> 结尾</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Cursor section */}
            <section id="section-cursor" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Cursor 配置</h2>
              <p className="text-sm text-muted-foreground mb-4">
                <a href="https://www.cursor.com" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">Cursor</a> 通过覆盖 OpenAI Base URL 接入网关。<strong>网关内的 Claude / GPT / Gemini / GLM 全系模型都通过 OpenAI 协议提供</strong>，不需要也无法走 Anthropic 协议（Cursor 当前版本未开放 Anthropic Base URL 自定义入口）。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">1. 覆盖 OpenAI Base URL</h3>
              <p className="text-sm text-muted-foreground mb-3">
                打开 Cursor → <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">Settings</code> → <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">Models</code>，找到 <strong>OpenAI API Key</strong> 区域：
              </p>
              <CodeBlock lang="TEXT" code={cursorOpenAISettingsCode} blockId="cursor-openai" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p>Cursor 设置页里也能看到 <strong>Anthropic API Key</strong> 区域，但<strong>没有 Override Anthropic Base URL 选项</strong>，无法把 Anthropic 流量指到自定义端点，因此本指南<strong>不使用</strong> Anthropic 方式。Claude 系列模型通过上面的 OpenAI Override 走网关即可，网关会自动做协议转换。</p>
                </div>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">2. 添加自定义模型</h3>
              <p className="text-sm text-muted-foreground mb-3">
                模型列表里点 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">+ Add Model</code>，输入模型名后保存。
              </p>

              <div className="flex gap-2 bg-amber-50 border border-amber-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-amber-400 rounded-full flex-shrink-0" />
                <div className="text-sm text-amber-800">
                  <p><strong>⚠️ Cursor 必须给模型名加 <code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">gw/</code> 前缀</strong></p>
                  <p className="mt-1">否则 Cursor 会把请求按官方 OpenAI 模型名拦截/改写，导致 Verify 失败或调不到对应模型。例如填 <code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">gw/claude-sonnet-4-6</code>，<strong>不要</strong>填 <code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">claude-sonnet-4-6</code>。</p>
                  <p className="mt-1">网关已兼容 <code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">gw/</code> 前缀，会自动剥离前缀后路由到对应模型。</p>
                </div>
              </div>

              <CodeBlock lang="TEXT" code={cursorAddModelCode} blockId="cursor-add-model" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">3. 推荐模型</h3>
              <p className="text-sm text-muted-foreground mb-3">
                Cursor 中所有模型名都要带 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">gw/</code> 前缀：
              </p>
              <CodeBlock lang="BASH" code={cursorModelsCode} blockId="cursor-models" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">4. 验证配置</h3>
              <p className="text-sm text-muted-foreground mb-3">
                配置完成后，在 Cursor 中打开任意文件，使用 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">Ctrl+L</code>（Mac: <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">Cmd+L</code>）打开 AI 对话框，选择已配置的模型发送一条消息，确认能正常收到回复。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2 mt-6">常见问题</h3>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">现象</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">原因 / 解决</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">Verify 一直转圈 / 失败</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">① Base URL 是否带 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">/v1</code> 且填在 OpenAI 区域；② 自定义模型名是否加了 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">gw/</code> 前缀</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">添加自定义模型后没响应</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">模型名忘了加 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">gw/</code> 前缀；Cursor 会按官方 OpenAI 模型名处理，导致请求被改写</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">Cursor Agent 模式偶发 400</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Cursor Agent 会发 Anthropic 形状请求体到 OpenAI 端点，本网关已兼容；如仍失败请反馈日志</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 text-foreground">想直接配 Anthropic 端点</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Cursor 当前版本没有 Anthropic Base URL 覆盖入口，统一走 OpenAI Override 即可</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Copilot CLI section */}
            <section id="section-copilot-cli" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Copilot CLI 配置</h2>
              <p className="text-sm text-muted-foreground mb-4">
                <a href="https://github.com/github/copilot-cli" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">GitHub Copilot CLI</a> 通过 BYOK（Bring Your Own Key）模式，可将推理流量切换到自定义的 OpenAI 或 Anthropic 兼容端点。本网关推荐使用 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">type=anthropic</code> 配置，覆盖网关内全部聊天模型。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">1. 安装 Copilot CLI</h3>
              <p className="text-sm text-muted-foreground mb-3">
                参考 GitHub 官方文档安装。常见的安装方式是 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">npm install -g @github/copilot</code> 或通过 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">gh extension install github/gh-copilot</code> 启用 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">gh copilot</code> 子命令。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">2. 配置环境变量（推荐 Anthropic 协议）</h3>
              <CodeBlock lang="BASH" code={copilotEnvCode} blockId="copilot-env" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p><strong>COPILOT_PROVIDER_TYPE=anthropic</strong> — 推荐配置。Copilot CLI 将走 Anthropic Messages 协议（<code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">/v1/messages</code>），网关内部按需在 Anthropic 与 OpenAI / Gemini 上游间转换，所有聊天模型均可使用。</p>
                  <p className="mt-1"><strong>COPILOT_PROVIDER_BASE_URL</strong> — 网关地址，<strong>不要</strong>以 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">/v1</code> 结尾。</p>
                  <p className="mt-1"><strong>COPILOT_PROVIDER_API_KEY</strong> — 在控制台「API 密钥」页面创建。</p>
                  <p className="mt-1"><strong>COPILOT_PROVIDER_MAX_PROMPT_TOKENS</strong> — 系统提示与工具定义本身就会占用约 21k token，建议至少设为 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">32000</code>，否则工具上下文会被截断。实际上限以所选模型为准。</p>
                </div>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">3. 可用模型</h3>
              <CodeBlock lang="BASH" code={copilotModelsCode} blockId="copilot-models" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">4. 启动使用</h3>
              <CodeBlock lang="BASH" code={copilotRunCode} blockId="copilot-run" copiedId={copiedId} onCopy={handleCopy} />

              <DocImage
                src={`${TOS_IMG_BASE}/copilot%E5%90%AF%E5%8A%A8%E6%95%88%E6%9E%9C.png`}
                alt="Copilot CLI 启动效果"
                caption="Copilot CLI 启动后通过网关回答的实际效果"
              />

              <h3 className="text-sm font-semibold text-foreground mb-2">5. 持久化配置</h3>
              <CodeBlock lang="BASH" code={copilotPersistCode} blockId="copilot-persist" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">备选方案：OpenAI 协议</h3>
              <p className="text-sm text-muted-foreground mb-3">
                若上游确认走 OpenAI 兼容端点，也可使用 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">type=openai</code>。注意 base URL 此时需带 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">/v1</code>。
              </p>
              <CodeBlock lang="BASH" code={copilotOpenAITypeCode} blockId="copilot-openai" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex gap-2 bg-amber-50 border border-amber-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-amber-400 rounded-full flex-shrink-0" />
                <div className="text-sm text-amber-800">
                  <p><strong>注意事项</strong></p>
                  <p className="mt-1">Anthropic 原生特性（Extended Thinking、Prompt Caching）只在 Claude 上游模型 + <code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">type=anthropic</code> 下完整生效；选择 OpenAI / Gemini 上游模型时这些字段会被网关安全降级，但基础对话与工具调用不受影响。</p>
                </div>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Trae section */}
            <section id="section-trae" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Trae 配置</h2>
              <p className="text-sm text-muted-foreground mb-4">
                <a href="https://www.trae.ai" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">Trae</a>（字节跳动）底层使用 OpenAI Chat Completions 协议。在 Trae 设置里添加自定义 Provider 即可接入网关，使用网关中所有聊天模型。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">1. 添加 Provider</h3>
              <CodeBlock lang="TEXT" code={traeUiCode} blockId="trae-ui" copiedId={copiedId} onCopy={handleCopy} />

              <DocImage
                src={`${TOS_IMG_BASE}/Trae%E6%A8%A1%E5%9E%8B%E9%85%8D%E7%BD%AE.png`}
                alt="Trae Settings 入口"
                caption="Trae Settings → Models 入口 — 侧边栏聊天框右上角齿轮"
              />
              <DocImage
                src={`${TOS_IMG_BASE}/Trae%E6%B7%BB%E5%8A%A0%E6%A8%A1%E5%9E%8B.png`}
                alt="Trae Add Model 弹窗"
                caption="Trae Add Model 弹窗 — Provider 选 OpenAI 或 OpenAI Compatible，按下方表格填写"
              />

              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">字段</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">值</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">API Key</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">sk-your-gateway-key</code></TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">Base URL</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">https://your-domain.com/v1</code>（<strong>带</strong> <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">/v1</code>）</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">Model</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">claude-sonnet-4-6</code> 或网关中其他模型名</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">2. 验证</h3>
              <p className="text-sm text-muted-foreground mb-3">
                在 Trae 聊天框输入「介绍你自己」并发送，能正常返回即接入成功。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">3. 可用模型</h3>
              <CodeBlock lang="BASH" code={traeModelsCode} blockId="trae-models" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2 mt-6">常见问题</h3>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">现象</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">原因 / 解决</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">添加 Provider 时找不到 Base URL 输入框</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">选择 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">OpenAI Compatible</code>（部分版本叫 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">Custom</code>），内置的 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">OpenAI</code> 项 Base URL 可能锁死</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">401 / 模型未找到</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Base URL 别忘记 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">/v1</code> 后缀；模型名与网关「模型管理」页保持一致</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 text-foreground">改完配置不生效</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">保存后回到聊天框右上角重新选一次刚添加的模型</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* OpenCode section */}
            <section id="section-opencode" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">OpenCode 配置</h2>
              <p className="text-sm text-muted-foreground mb-4">
                <a href="https://github.com/sst/opencode" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">OpenCode</a> 是 sst 推出的 AI 编程 CLI，支持 OpenAI 兼容 API。通过其内置的 <strong>TUI 设置界面</strong>添加 Provider 接入网关，<strong>无需编辑配置文件</strong>。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">1. 在 TUI 中添加 Provider</h3>
              <CodeBlock lang="TEXT" code={opencodeAddProviderCode} blockId="opencode-add-provider" copiedId={copiedId} onCopy={handleCopy} />

              <DocImage
                src={`${TOS_IMG_BASE}/opencode%E6%96%B0%E5%A2%9E%E6%8F%90%E4%BE%9B%E5%95%86.png`}
                alt="OpenCode 设置入口"
                caption="OpenCode 命令面板 — 选择 Settings / Providers，进入新增 Provider"
              />
              <DocImage
                src={`${TOS_IMG_BASE}/opencode%E8%87%AA%E5%AE%9A%E4%B9%89%E6%8F%90%E4%BE%9B%E5%95%86.png`}
                alt="OpenCode 选择 OpenAI Compatible"
                caption="Provider 类型选 OpenAI Compatible，按下方表格填入网关信息"
              />

              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">字段</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">值</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">Name</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">LLM Gateway</code>（自定义）</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">Base URL</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">https://your-domain.com/v1</code>（<strong>带</strong> <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">/v1</code>）</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 font-mono text-xs">API Key</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">sk-your-gateway-key</code></TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">Model</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground"><code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">claude-sonnet-4-6</code> 或网关中其他模型名</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">2. 选择模型并使用</h3>
              <p className="text-sm text-muted-foreground mb-3">
                在 OpenCode 主界面按 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">/model</code>（或对应快捷键）打开模型选择面板，挑选 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">LLM Gateway</code> 下的模型即可：
              </p>
              <DocImage
                src={`${TOS_IMG_BASE}/opencode%E6%A8%A1%E5%9E%8B%E9%80%89%E6%8B%A9.png`}
                alt="OpenCode 模型选择 TUI"
                caption="OpenCode 模型选择面板 — 展开 LLM Gateway 分组挑选具体模型"
              />
              <CodeBlock lang="BASH" code={opencodeRunCode} blockId="opencode-run" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2 mt-6">常见问题</h3>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">现象</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">原因 / 解决</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">找不到 Add Provider 入口</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">不同版本菜单名不同，可能叫 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">Providers</code> / <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">Models</code> / <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">Custom Endpoints</code>，关键词搜「OpenAI Compatible」</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground">测试连接失败</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">Base URL 是否带 <code className="bg-muted px-1 py-0.5 rounded text-xs font-mono">/v1</code>；API Key 是否填写正确</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 text-foreground">模型选择器里看不到刚加的 Provider</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">重启 OpenCode；或在设置里把该 Provider 切到「启用」状态</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Gemini API section */}
            <section id="section-gemini-api" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">Gemini API</h2>
              <p className="text-sm text-muted-foreground mb-4">
                Gateway 提供 Google Gemini 原生 API 的透传代理，支持 v1 和 v1beta 两个版本。可直接使用 Gemini SDK 或原生 REST 调用。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">端点格式</h3>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-4">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">Action</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">端点</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-muted-foreground">生成内容</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /gemini/v1/models/{"{model}"}:generateContent</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-muted-foreground">流式生成</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /gemini/v1/models/{"{model}"}:streamGenerateContent</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-muted-foreground">生成内容（v1beta）</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /gemini/v1beta/models/{"{model}"}:generateContent</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-muted-foreground">流式生成（v1beta）</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /gemini/v1beta/models/{"{model}"}:streamGenerateContent</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-muted-foreground">缓存内容</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /gemini/v1/models/{"{model}"}/cachedContents</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 text-muted-foreground">缓存内容（v1beta）</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">POST /gemini/v1beta/models/{"{model}"}/cachedContents</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>

              <div className="flex gap-2 bg-amber-50 border border-amber-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-amber-400 rounded-full flex-shrink-0" />
                <div className="text-sm text-amber-800">
                  <p>图像生成模型（如 <code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">gemini-3.1-flash-image-preview</code>）需要使用 <code className="bg-amber-100 px-1 py-0.5 rounded text-xs font-mono">v1beta</code> 版本端点。</p>
                </div>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">文本生成</h3>
              <CodeBlock lang="BASH" code={geminiApiCurl} blockId="gemini-curl" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">流式生成</h3>
              <CodeBlock lang="BASH" code={geminiStreamCurl} blockId="gemini-stream-curl" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">图像生成（v1beta）</h3>
              <p className="text-sm text-muted-foreground mb-3">
                通过 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">responseModalities</code> 指定返回 IMAGE，模型会在响应中返回 base64 编码的图片数据。按次计费。
              </p>
              <CodeBlock lang="BASH" code={geminiImageCurl} blockId="gemini-image-curl" copiedId={copiedId} onCopy={handleCopy} />

              <h3 className="text-sm font-semibold text-foreground mb-2">Python SDK 示例</h3>
              <CodeBlock lang="PYTHON" code={geminiPythonSdk} blockId="gemini-python-sdk" copiedId={copiedId} onCopy={handleCopy} />

              <div className="flex gap-2 bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-primary rounded-full flex-shrink-0" />
                <div className="text-sm text-blue-800">
                  <p>请求和响应格式与 Google Gemini API 完全一致，可参考 <a href="https://ai.google.dev/api/generate-content" target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">Gemini API 官方文档</a>。</p>
                  <p className="mt-1">使用 Gemini SDK 时，将 API endpoint 设为 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">https://your-domain.com/gemini</code>，API Key 使用网关 Key。SDK 会自动拼接 <code className="bg-blue-100 px-1 py-0.5 rounded text-xs font-mono">/v1beta/models/...</code> 路径。</p>
                </div>
              </div>
            </section>


            <div className="border-t border-border mb-8" />

            {/* Streaming & Timeout section (REQ-PO-29) */}
            <section id="section-streaming" className="mb-8">
              <h2 className="text-lg font-bold text-foreground mb-2">流式响应与超时</h2>
              <p className="text-sm text-muted-foreground mb-4">
                Gateway 所有 LLM 端点均支持 Server-Sent Events (SSE) 流式响应。以下是流式行为说明和推荐的超时配置。
              </p>

              <h3 className="text-sm font-semibold text-foreground mb-2">SSE 流式行为</h3>
              <div className="space-y-2 mb-5">
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>Chat Completions: 设置 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">stream: true</code>，响应为 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">text/event-stream</code> 格式，每个 chunk 以 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">data: </code> 前缀发送，最后以 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">data: [DONE]</code> 结束</span>
                </div>
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>Messages API: 设置 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">stream: true</code>，响应为 Anthropic SSE 格式，包含 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">message_start</code>、<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">content_block_delta</code>、<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">message_stop</code> 等事件</span>
                </div>
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>Responses API: 默认流式，响应包含 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">response.created</code>、<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">response.output_text.delta</code>、<code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">response.completed</code> 等事件</span>
                </div>
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>Gemini API: 使用 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">:streamGenerateContent?alt=sse</code> 端点获取流式响应</span>
                </div>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">推荐超时配置</h3>
              <div className="bg-card border border-border rounded-xl overflow-hidden mb-5">
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/30">
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">场景</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">推荐超时</TableHead>
                      <TableHead className="text-left px-4 py-2.5 font-medium text-muted-foreground">说明</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground text-xs font-medium">非流式请求</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">120-300s</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground text-xs">大模型生成可能较慢，建议至少 2 分钟</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground text-xs font-medium">流式请求（首 token）</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">30-60s</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground text-xs">首个 token 到达时间，超过可能是上游异常</TableCell>
                    </TableRow>
                    <TableRow className="border-b border-border">
                      <TableCell className="px-4 py-2.5 text-foreground text-xs font-medium">流式请求（chunk 间隔）</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">30s</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground text-xs">两个 chunk 之间的最大间隔</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="px-4 py-2.5 text-foreground text-xs font-medium">图像生成</TableCell>
                      <TableCell className="px-4 py-2.5 font-mono text-xs">120s</TableCell>
                      <TableCell className="px-4 py-2.5 text-muted-foreground text-xs">图像生成耗时较长</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>

              <h3 className="text-sm font-semibold text-foreground mb-2">重试策略</h3>
              <div className="space-y-2 mb-5">
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>429 (Too Many Requests): 按 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">Retry-After</code> 头等待后重试，或使用指数退避（1s → 2s → 4s）</span>
                </div>
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>500 / 502 / 503: 可安全重试，建议最多 3 次，使用指数退避</span>
                </div>
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>400 / 401 / 402 / 403: 不可重试，需修正请求或充值后再试</span>
                </div>
                <div className="flex gap-2 text-sm text-muted-foreground">
                  <span className="text-primary">&#8226;</span>
                  <span>流式中断: 如果流式响应中途断开，不建议自动重试（可能导致重复计费），应提示用户重新发送</span>
                </div>
              </div>

              <div className="flex gap-2 bg-amber-50 border border-amber-200 rounded-xl p-4 mb-5">
                <div className="w-1 bg-amber-400 rounded-full flex-shrink-0" />
                <div className="text-sm text-amber-800">
                  Gateway 的默认请求超时为 5 分钟。如果你的请求涉及大量 token 生成（如长文档），建议使用流式模式以避免超时。
                </div>
              </div>
            </section>

            <div className="border-t border-border mb-8" />

            {/* Errors section */}
            <section id="section-errors" className="mb-12">
              <h2 className="text-lg font-bold text-foreground mb-4">错误处理</h2>
              <div className="flex flex-col gap-3">
                {errorsContent.map((e) => (
                  <div key={e.code} className="flex items-center gap-4 bg-card border border-border rounded-xl px-5 py-4 shadow-card">
                    <div className="w-14 flex-shrink-0">
                      <span className={`text-sm font-mono font-bold ${
                        e.code.startsWith("4") ? "text-destructive" : "text-amber-600"
                      }`}>{e.code}</span>
                    </div>
                    <div className="w-px h-8 bg-border flex-shrink-0" />
                    <div>
                      <div className="text-sm font-semibold text-foreground">{e.title}</div>
                      <div className="text-xs text-muted-foreground mt-0.5">{e.desc}</div>
                    </div>
                  </div>
                ))}
              </div>

              <h3 className="text-base font-semibold text-foreground mt-6 mb-3">错误码字典</h3>
              <p className="text-sm text-muted-foreground mb-3">
                每个错误响应包含 <code className="text-xs bg-muted px-1.5 py-0.5 rounded">error.code</code> 字段，可用于程序化错误处理。
              </p>
              <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
                <Table className="w-full">
                  <TableHeader>
                    <TableRow className="table-header">
                      <TableHead className="text-left px-5 py-3 text-xs">错误码</TableHead>
                      <TableHead className="text-left px-5 py-3 text-xs">说明</TableHead>
                      <TableHead className="text-left px-5 py-3 text-xs">可重试</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {errorCodes.map((ec) => (
                      <TableRow key={ec.code} className="border-t border-border">
                        <TableCell className="px-5 py-2.5 text-xs font-mono text-foreground">{ec.code}</TableCell>
                        <TableCell className="px-5 py-2.5 text-xs text-muted-foreground">{ec.desc}</TableCell>
                        <TableCell className="px-5 py-2.5 text-xs">
                          {ec.retryable ? (
                            <span className="text-emerald-600">是</span>
                          ) : (
                            <span className="text-muted-foreground">否</span>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </section>
          </div>

          <div className="border-t border-border py-6 text-center">
            <BeianBar />
          </div>
        </div>
      </div>
    </div>
  );
};

export default DocsPage;
