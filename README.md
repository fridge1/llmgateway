# LLM Gateway

LLM Gateway 是一个统一的大模型 API 网关与商业化运营平台。它把 OpenAI、Anthropic、Google Gemini 等协议入口汇聚到同一套认证、路由、计费、订阅、租户和管理后台中，让 Claude Code、Codex CLI、Cursor、Windsurf、Aider 等主流 AI 编码工具用同一个网关地址和 API Key 即可访问已配置的全部模型。

项目由 Go 后端、React Web 控制台、Tauri 桌面端和 PostgreSQL 数据库组成。模型、上游、定价、套餐、租户、订单等业务数据由数据库驱动，YAML 配置只保留服务运行和第三方集成所需的基础参数。

## 核心能力

- **多协议 LLM 代理**：支持 OpenAI Chat Completions、OpenAI Responses、Anthropic Messages、Gemini 原生 API，可在 OpenAI / Anthropic / Gemini 协议间转换。
- **模型路由与故障转移**：模型配置来自数据库，支持模型别名、`gw/` 前缀、上游协议标记、模型覆盖、加权/优先路由、熔断器和路由热重载。
- **统一认证与访问控制**：支持用户 API Key、租户 API Key、租户子用户 API Key，按用户状态、套餐范围、租户权限和模型权限控制访问。
- **计费与订阅**：按 token、缓存 token 或图片张数计费，支持余额扣费、租户余额、订阅套餐额度、API Key 用量统计、异步结算和低余额通知。
- **商业化运营**：内置支付宝充值、订单、发票抬头与开票申请、充值赠送、邀请奖励、签到、任务、公告、通知和审计日志。
- **企业租户**：支持企业租户、成员邀请、角色权限、租户专属 API Key、子用户账号、租户流水、用量分析和专属定价/折扣。
- **AI 工具产品层**：内置在线 Playground、翻译工具、图片生成/编辑任务、图片分享入口，以及可一键配置本机 AI 工具的桌面端。
- **Web 与桌面控制台**：Web 覆盖官网、用户控制台、管理员后台与组织子账号后台；桌面端覆盖登录、余额、Key、用量、套餐、租户和工具配置。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.26 · 标准库 HTTP Server · golang-migrate |
| 数据库 | PostgreSQL 16 |
| Web 前端 | React 19 · TypeScript · Vite · Tailwind CSS |
| 桌面端 | Tauri 2 · Rust · React · TypeScript |
| 支付 | 支付宝网页支付 / WAP 支付 / 异步通知 |
| 短信 | 火山引擎 SMS |
| 对象存储 | 火山引擎 TOS |

## 快速开始

### 方式一：本地 Docker Compose（推荐体验）

本地 Compose 自带 PostgreSQL 容器，零配置即可启动完整服务。

```bash
docker compose -f docker-compose.local.yml up --build -d
```

启动后访问：

- Web 控制台：http://localhost:3000
- 健康检查：http://localhost:3000/health
- 首次注册时填入初始化 Token（默认 `your-init-token`）即可成为管理员

常用命令：

```bash
docker compose -f docker-compose.local.yml ps
docker compose -f docker-compose.local.yml logs -f backend
docker compose -f docker-compose.local.yml down      # 停止（保留数据）
docker compose -f docker-compose.local.yml down -v   # 停止并清除数据
```

### 方式二：源码开发

前置条件：Go ≥ 1.26、Node.js ≥ 18、PostgreSQL ≥ 15

```bash
# 1. 准备配置
cp config.example.yaml config.yaml
# 编辑 config.yaml：database.dsn、admin.jwt_secret、admin.init_token 等

# 2. 启动后端（默认端口 9090）
go run ./cmd/gateway -config config.yaml -migrations ./migrations

# 3. 启动前端（端口 5173，代理到后端 :9090）
cd web && npm install && npm run dev
```

### 方式三：生产 Docker Compose

生产环境通过 `.env` 文件注入敏感信息，配置模板见 `.env.example`。

```bash
cp .env.example .env   # 填入实际密钥
docker compose up --build -d
```

至少需要配置以下环境变量：

- `JWT_SECRET`、`POSTGRES_PASSWORD`、`ADMIN_INIT_TOKEN`
- `SMS_ACCESS_KEY`、`SMS_SECRET_KEY`、`SMS_SIGN_NAME`、`SMS_ACCOUNT` 及短信模板 ID
- `ALIPAY_APP_ID`、`ALIPAY_NOTIFY_URL`、`ALIPAY_RETURN_URL`、`ALIPAY_IS_PRODUCTION`
- 启用图片/PPT/发票能力时还需配置 `TOS_*` 对象存储参数

## API 接口

### LLM 兼容接口

| 端点 | 格式 | 典型用途 |
|---|---|---|
| `GET /v1/models` | OpenAI 兼容模型列表 | 工具发现可用模型 |
| `GET /v1/balance` | 网关扩展 | API Key 查询余额 |
| `POST /v1/chat/completions` | OpenAI Chat Completions | Cursor、Windsurf、Aider、OpenAI SDK |
| `POST /v1/responses` | OpenAI Responses | Codex CLI、Responses API 客户端 |
| `POST /v1/messages` | Anthropic Messages | Claude Code、Anthropic SDK |
| `POST /v1/messages/count_tokens` | Anthropic token 统计 | Claude 兼容工具 |
| `POST /gemini/v1/models/{model}:generateContent` | Gemini 原生 | Gemini SDK |
| `POST /gemini/v1beta/models/{model}:streamGenerateContent` | Gemini SSE | Gemini 流式输出 |
| `POST /v1/images/generations` | OpenAI 图像生成兼容 | 同步图片生成 |
| `POST /v1/images/edits` | OpenAI 图像编辑兼容 | 同步图片编辑 |
| `POST /v1/images/tasks` | 网关异步图片任务 | 任务式生成/轮询 |

认证使用 `Authorization: Bearer <API_KEY>`；Anthropic Messages 入口也支持 `x-api-key: <API_KEY>`。

### 控制台 API

- **认证**：`/api/login`、`/api/register`、`/api/sms/send`、`/api/reset-password`
- **用户**：`/api/keys`、`/api/billing/*`、`/api/subscription/*`、`/api/payment/*`
- **模型**：`/api/models`、`/api/pricing`
- **租户**：`/api/tenants/*`、`/api/sub-user/*`
- **工具**：`/api/image/*`、`/api/ppt/*`、`/api/chat/*`、`/api/invoice/*`
- **管理后台**：`/api/admin/*`（模型、上游、定价、用户、订单、套餐、租户、发票、审计等）

## 配置说明

模型、上游、定价等业务配置保存在数据库中，通过管理后台修改，变更后路由器自动重建，无需重启。`config.yaml` 仅负责运行时基础设置：

| 配置项 | 说明 |
|---|---|
| `server.port` | 后端监听端口，默认 `9090` |
| `server.request_timeout` | 上游请求超时 |
| `server.max_request_body_bytes` | 请求体大小限制 |
| `server.no_pricing_strategy` | 无定价模型策略：`reject` / `warn` / `allow` |
| `server.rate_limit.*` | 按 IP、用户、Key 的限流 |
| `admin.jwt_secret` | JWT 签名密钥，生产环境必须替换 |
| `admin.init_token` | 首个管理员注册所需 Token |
| `database.dsn` | PostgreSQL 连接串，支持环境变量展开 |
| `payment.alipay.*` | 支付宝配置 |
| `sms.*` | 短信服务配置 |
| `tos.*` | 对象存储配置 |
| `promotion.*` | 注册赠送、首充赠送、邀请奖励 |
| `retention.*` | 低余额提醒、签到奖励等留存策略 |

## 目录结构

```text
cmd/gateway/                 # 主服务入口与 HTTP 路由注册
cmd/compensate-referral/     # 邀请奖励补偿工具
internal/proxy/              # OpenAI Chat 代理核心、认证、故障转移、流式转发
internal/anthropic/          # Anthropic Messages 兼容层
internal/responses/          # OpenAI Responses 兼容层
internal/gemini/             # Gemini 原生 API 兼容层
internal/adapter/            # 协议间请求转换
internal/router/             # 模型到上游池的路由表
internal/balancer/           # 上游负载均衡
internal/circuit/            # 熔断器
internal/billing/            # 计费、订阅用量、结算
internal/pricing/            # 模型定价缓存与管理
internal/subscription/       # 套餐、额度、订阅订单
internal/payment/            # 支付宝支付与异步通知
internal/store/              # PostgreSQL 数据访问层
internal/admin/              # 管理后台接口
internal/image/              # 图片生成/编辑
internal/ppt/                # PPT 生成任务
internal/invoice/            # 发票管理
migrations/                  # SQL 迁移文件
web/                         # React Web 应用与 Nginx 配置
desktop/                     # Tauri 桌面端
scripts/                     # 运维与数据修复脚本
```

## 开发与测试

```bash
# 后端测试
go test ./...

# Web 构建与 lint
cd web && npm run build && npm run lint

# 桌面端构建
cd desktop && npm run tauri build

# 健康检查
curl http://localhost:9090/health
curl http://localhost:9090/v1/models
```

## 安全注意事项

- 切勿提交 `.env`、真实 API Key、JWT 密钥、数据库密码、支付宝私钥或对象存储密钥。
- 生产环境必须替换 `admin.jwt_secret`，并设置非默认的 `admin.init_token`。
- 支付宝异步通知需配置公网可访问的 HTTPS URL，并确保支付宝后台与 `notify_url` 一致。
- 图片、PPT 和发票能力依赖对象存储；缺少 TOS 配置时相关功能会降级或不可用。

## License

本项目采用 [PolyForm Noncommercial License 1.0.0](LICENSE) 授权。

仅供个人学习、研究和内部非商业用途使用，**禁止任何形式的商业使用**，包括但不限于销售、付费托管、SaaS 转售等。详见 [LICENSE](LICENSE) 文件。
