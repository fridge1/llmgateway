# LLM Gateway

LLM Gateway 是一个面向开发者和企业团队的大模型 API 网关与商业化运营平台。它把 OpenAI、Anthropic、Google Gemini 等协议入口统一到同一套认证、路由、计费、订阅、租户和管理后台中，让 Claude Code、Codex CLI、Cursor、Windsurf、Aider 等工具可以使用同一个网关地址和 API Key 访问已配置的模型。

项目由 Go 后端、React Web 控制台、Tauri 桌面端和 PostgreSQL 数据库组成。模型、上游、定价、套餐、租户、订单、发票、图片任务等业务数据主要由数据库驱动，YAML 配置只保留服务运行和第三方集成所需的基础参数。

## 核心能力

- **多协议 LLM 代理**：支持 OpenAI Chat Completions、OpenAI Responses、Anthropic Messages、Gemini 原生 API，并在 OpenAI/Anthropic/Gemini 协议间转换请求和响应。
- **模型路由与故障转移**：模型配置来自数据库，支持 display name、`gw/` 前缀、上游协议标记、模型覆盖、加权/优先路由、熔断器和路由热重载。
- **统一认证与访问控制**：支持用户 API Key、租户 API Key、租户子用户 API Key，按用户状态、套餐范围、租户权限和模型权限控制访问。
- **计费与订阅**：按 token、缓存 token 或图片张数计费，支持余额扣费、租户余额、订阅套餐额度、API Key 用量统计、异步结算和低余额通知。
- **商业化运营**：内置支付宝充值、订单、发票抬头和开票申请、充值赠送、邀请奖励、签到、任务、抽奖、公告、通知和审计日志。
- **企业租户**：支持企业租户、成员邀请、角色权限、租户专属 API Key、子用户账号、租户流水、用量分析和租户专属定价/折扣。
- **AI 工具产品层**：提供在线 Playground、翻译工具、图片生成/编辑任务、PPT 生成任务、图片分享独立入口，以及可配置本机 AI 工具的桌面端。
- **Web 与桌面控制台**：Web 覆盖官网、用户控制台、管理员后台、组织子账号后台；Tauri 桌面端覆盖登录、余额、Key、用量、套餐、租户和工具配置。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.26 · 标准库 HTTP Server · golang-migrate |
| 数据库 | PostgreSQL 16 |
| Web 前端 | React 19 · TypeScript · Vite · Tailwind CSS · shadcn/ui 风格组件 |
| 桌面端 | Tauri 2 · Rust · React · TypeScript |
| 支付 | 支付宝网页支付 / WAP 支付 / 异步通知 |
| 短信 | 火山引擎 SMS |
| 对象存储 | 火山引擎 TOS，用于图片和发票/PPT 相关文件 |

## 主要接口

### LLM 兼容接口

| 端点 | 格式 | 典型用途 |
|---|---|---|
| `GET /v1/models` | OpenAI 兼容模型列表 | 工具发现可用模型 |
| `GET /v1/balance` | 网关扩展接口 | API Key 查询余额 |
| `POST /v1/chat/completions` | OpenAI Chat Completions | Cursor、Windsurf、Aider、OpenAI SDK |
| `POST /v1/responses` | OpenAI Responses | Codex CLI、Responses API 客户端 |
| `GET /v1/responses/{id}` | OpenAI Responses 查询占位 | 返回 JSON 格式 404，不持久化响应 |
| `POST /v1/messages` | Anthropic Messages | Claude Code、Anthropic SDK |
| `POST /v1/messages/count_tokens` | Anthropic token 统计 | Claude 兼容工具 |
| `POST /gemini/v1/models/{model}:generateContent` | Gemini 原生 | Gemini SDK |
| `POST /gemini/v1beta/models/{model}:streamGenerateContent` | Gemini SSE | Gemini 流式输出 |
| `POST /v1/images/generations` | OpenAI 图像生成兼容 | 同步图片生成 |
| `POST /v1/images/edits` | OpenAI 图像编辑兼容 | 同步图片编辑 |
| `POST /v1/images/tasks` | 网关异步图片任务 | 任务式生成/轮询 |

认证使用 `Authorization: Bearer <API_KEY>`；Anthropic Messages 兼容入口也支持 `x-api-key: <API_KEY>`。

### 控制台 API

- `/api/login`、`/api/register`、`/api/sms/send`、`/api/reset-password`：手机号、短信验证码、JWT Cookie 登录体系。
- `/api/keys`、`/api/billing/*`、`/api/subscription/*`、`/api/payment/*`：用户 API Key、余额、交易、套餐和充值。
- `/api/models`、`/api/pricing`：登录用户可见的模型与定价信息；模型写操作用于管理配置。
- `/api/tenants/*`、`/api/sub-user/*`、`/org/*` 前端路由：企业租户、成员、子用户和组织用量。
- `/api/image/*`、`/api/ppt/*`、`/api/chat/*`、`/api/invoice/*`：图片任务、PPT 任务、聊天会话和发票流程。
- `/api/admin/*`：管理员后台，包括模型、上游、定价、用户、订单、套餐、租户、发票、公告、抽奖、审计和统计。

## 运行方式

### 方式一：本地 Docker Compose

本地 Compose 自带 PostgreSQL，适合快速体验完整服务。

```bash
docker compose -f docker-compose.local.yml up --build -d
```

默认访问：

- Web 控制台：http://localhost:3000
- 健康检查：http://localhost:3000/health
- 首次注册管理员初始化 Token：`your-init-token`

常用命令：

```bash
docker compose -f docker-compose.local.yml ps
docker compose -f docker-compose.local.yml logs -f backend
docker compose -f docker-compose.local.yml down
docker compose -f docker-compose.local.yml down -v  # 同时清空本地数据库 volume
```

### 方式二：源码开发

前置条件：

- Go >= 1.26
- Node.js >= 18
- PostgreSQL >= 15

准备配置：

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml：database.dsn、admin.jwt_secret、admin.init_token 等
```

启动后端：

```bash
go run ./cmd/gateway -config config.yaml -migrations ./migrations
```

后端默认监听 `:9090`，启动时会自动执行 `migrations/` 下的数据库迁移。

启动 Web 前端：

```bash
cd web
npm install
npm run dev
```

启动桌面端开发环境：

```bash
cd desktop
npm install
npm run tauri dev
```

## 生产部署

生产 Docker Compose 使用 `docker-compose.yml`：

- `backend`：Go 后端容器，读取 `config.docker.yaml`，内部监听 `9090`，不直接暴露到宿主机。
- `web`：Nginx + Web 静态资源，监听宿主机 `80`，把 `/api/`、`/v1/`、`/gemini/` 等请求反向代理到后端。

部署命令：

```bash
docker compose down
docker compose up --build -d
docker compose ps
docker compose logs -f backend
```

生产环境需要通过 `.env` 注入至少以下变量：

- `JWT_SECRET`、`POSTGRES_PASSWORD`、`ADMIN_INIT_TOKEN`
- `SMS_ACCESS_KEY`、`SMS_SECRET_KEY`、`SMS_SIGN_NAME`、`SMS_ACCOUNT` 和短信模板 ID
- `ALIPAY_APP_ID`、`ALIPAY_NOTIFY_URL`、`ALIPAY_RETURN_URL`、`ALIPAY_IS_PRODUCTION`
- 如启用图片/PPT/发票文件能力，还需要配置 `tos.*` 对象存储参数

## 配置要点

`config.yaml` 主要负责运行时基础设置：

| 配置项 | 说明 |
|---|---|
| `server.port` | 后端监听端口，默认 `9090` |
| `server.request_timeout` | 上游请求超时，长文本/图片/PPT 场景通常需要较长时间 |
| `server.max_request_body_bytes` | 请求体大小限制，默认示例为 20MB |
| `server.no_pricing_strategy` | 无定价模型策略：`reject`、`warn`、`allow` |
| `server.rate_limit.*` | 按 IP、用户、Key 的限流配置 |
| `admin.jwt_secret` | JWT 签名密钥，生产环境必须使用强随机值 |
| `admin.init_token` | 首个管理员注册或管理员初始化所需 Token |
| `database.dsn` | PostgreSQL 连接串，支持环境变量展开 |
| `payment.alipay.*` | 支付宝 App ID、证书路径、回调和跳转 URL |
| `sms.*` | 火山引擎短信配置 |
| `promotion.*` | 注册赠送、首充赠送、邀请奖励等增长配置 |
| `retention.*` | 低余额提醒、周报、沉默用户召回、签到奖励 |
| `tos.*` | 火山引擎 TOS 对象存储配置 |
| `billing.*` | 异步计费 worker、队列和溢出并发配置 |

模型、上游、定价、套餐、租户专属定价等业务配置保存在数据库中，通过管理后台和对应 API 修改。模型变更后后端会重建路由器，无需重启服务。

## 目录结构

```text
cmd/gateway/                 # 主服务入口、迁移、依赖装配、HTTP 路由注册
cmd/compensate-referral/     # 邀请奖励补偿工具
internal/adapter/            # OpenAI 与 Anthropic 请求转换
internal/anthropic/          # Anthropic Messages 兼容层
internal/responses/          # OpenAI Responses 兼容层
internal/gemini/             # Gemini 原生 API 兼容层
internal/proxy/              # OpenAI Chat 代理核心、认证、路由、故障转移、流式转发
internal/router/             # 模型到上游池的路由表
internal/balancer/           # 上游负载均衡
internal/circuit/            # 熔断器
internal/billing/            # 余额、租户、子用户、订阅用量计费
internal/pricing/            # 模型定价缓存和管理 API
internal/subscription/       # 套餐、额度、订阅订单和使用记录
internal/payment/            # 支付宝支付与异步通知
internal/admin/              # 登录注册、模型、用户、租户、公告、后台接口
internal/api/                # 租户子用户 API
internal/apikey/             # API Key 创建、缓存、批量 last_used 更新
internal/image/              # 图片生成/编辑同步接口与异步任务 worker
internal/imageshare/         # 图片分享 Key 与独立入口授权
internal/ppt/                # PPT 多 Agent 生成任务 worker
internal/invoice/            # 发票抬头、开票申请、管理员处理
internal/chat/               # 控制台聊天会话
internal/task/               # 成长任务
internal/checkin/            # 每日签到
internal/lottery/            # 抽奖活动
internal/rechargelottery/    # 充值抽奖
internal/notification/       # 站内通知
internal/retention/          # 低余额、周报、召回等留存逻辑
internal/store/              # PostgreSQL 数据访问层
internal/middleware/         # JWT、角色、租户、限流、CORS、安全头、日志、恢复
migrations/                  # SQL 迁移文件
web/                         # React Web 应用和 Nginx 配置
desktop/                     # Tauri 桌面端
docs/                        # 设计、部署、排障和工具配置文档
scripts/                     # 运维验证和数据修复脚本
```

## 开发与验证

后端测试：

```bash
go test ./...
```

Web 构建和 lint：

```bash
cd web
npm run build
npm run lint
```

桌面端构建：

```bash
cd desktop
npm run build
npm run tauri build
```

常用健康检查：

```bash
curl http://localhost:9090/health
curl http://localhost:9090/v1/models
```

## 安全注意事项

- 不要提交 `.env`、真实 API Key、JWT 密钥、数据库密码、支付宝私钥或对象存储密钥。
- 生产环境必须替换 `admin.jwt_secret`，并设置非默认 `admin.init_token`。
- 上游 URL 由代理核心校验协议和私网地址，避免把代理配置成内网 SSRF 通道。
- 支付宝异步通知必须配置公网可访问的 HTTPS URL，并确保支付宝后台配置与 `payment.alipay.notify_url` 一致。
- 图片、PPT 和发票文件能力依赖对象存储；缺少 TOS 配置时相关文件上传/生成能力会降级或不可用。

## License

Private
