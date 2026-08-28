# AGENTS.md

本文件为 Codex (Codex.ai/code) 在此代码库中工作时提供指导。

**重要：请始终使用中文与用户交互和回复。**

## 行为准则

通用编码准则：先思考再编码、简洁优先、精准修改、目标驱动执行。

## 项目概述

LLM Gateway 是一个统一的大语言模型 API 网关，支持多种协议格式（OpenAI、Anthropic、Google Gemini），可与 Codex、Cursor、Windsurf、Codex CLI 等 AI 编码工具无缝集成。

**技术栈：**
- 后端：Go 1.26，标准库 HTTP Server，PostgreSQL 16
- 前端：React 19，TypeScript，Vite，Tailwind CSS
- 数据库：PostgreSQL + golang-migrate
- 支付：支付宝
- 短信：火山引擎 SMS

## 开发命令

### 后端

```bash
# 启动后端（默认端口 9090）
go run ./cmd/gateway -config config.yaml

# 运行测试
go test ./...

# 构建二进制文件
go build -o gateway ./cmd/gateway/

# 使用自定义配置和迁移路径运行
go run ./cmd/gateway -config config.local.yaml -migrations ./migrations
```

### 前端

```bash
cd web

# 安装依赖
npm install

# 启动开发服务器（端口 5173，代理到后端 :9090）
npm run dev

# 生产构建
npm run build

# 运行 linter
npm run lint
```

### Docker 生产部署

**当前生产环境使用 Docker Compose 部署，配置文件为 `docker-compose.yml`。**

#### 部署架构

- **backend** 服务（容器名 `llm-gateway`）
  - 使用 `config.docker.yaml` 作为配置文件
  - 敏感信息通过 `.env` 文件注入环境变量
  - 9090 端口不对外暴露，仅内部网络访问
  - 配置健康检查和资源限制（4G 内存 / 4 CPU）
  
- **web** 服务（容器名 `llm-gateway-web`）
  - Nginx 反向代理，监听 80 端口
  - 所有 API 请求（`/api/`, `/v1/`, `/gemini/`, `/v3/`）转发到 backend
  - 前端 SPA 静态文件服务
  - 依赖 backend 健康检查通过后启动

#### 部署命令

```bash
# 完整重新部署（会短暂中断服务）
docker compose down
docker compose up --build -d

# 仅重启服务（不重新构建）
docker compose restart

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f backend
docker compose logs -f web
```

#### 配置文件说明

- **`docker-compose.yml`** - Docker Compose 编排配置
- **`config.docker.yaml`** - 后端配置模板（使用环境变量占位符）
- **`.env`** - 环境变量（包含所有敏感信息，不提交到 git）
- **`certs/`** - 支付宝证书目录（挂载到容器）

#### 环境变量清单

`.env` 文件需包含以下变量：
- `JWT_SECRET` - JWT 签名密钥
- `POSTGRES_PASSWORD` - 数据库密码
- `AUTH_TOKEN` - 网关认证 token
- `ADMIN_TOKEN` - 管理员 token
- `SMS_ACCESS_KEY`, `SMS_SECRET_KEY`, `SMS_SIGN_NAME`, `SMS_ACCOUNT` - 短信服务
- `SMS_TPL_LOGIN`, `SMS_TPL_REGISTER`, `SMS_TPL_RESET` - 短信模板 ID
- `ALIPAY_APP_ID`, `ALIPAY_NOTIFY_URL`, `ALIPAY_RETURN_URL`, `ALIPAY_IS_PRODUCTION` - 支付宝配置

#### 数据库配置

生产环境数据库位于 `${DB_HOST}:5432`（内网地址），在 `config.docker.yaml` 中配置：
```yaml
database:
  dsn: "postgres://gateway:${POSTGRES_PASSWORD}@${DB_HOST}:5432/gateway?sslmode=disable"
```

#### 验证部署

```bash
# 检查健康状态
curl http://localhost/health
# 预期输出: {"db":"ok","status":"ok"}

# 检查前端
curl -I http://localhost/
# 预期: HTTP 200

# 检查 API（需要认证）
curl http://localhost/v1/models
# 预期: HTTP 401（未认证）
```

#### 故障排查

如果 web 容器启动失败并报 `host not found in upstream "backend"`：
```bash
# 重启 web 容器（此时 backend 已注册到 DNS）
docker compose restart web
```

如果端口 80 被占用：
```bash
# 查找占用端口的容器
docker ps -a --filter "publish=80"

# 停止并删除旧容器
docker stop <container_id>
docker rm <container_id>
```

### Docker 本地部署

**本地开发/测试使用独立的 Docker Compose，配置文件为 `docker-compose.local.yml`。**

与生产部署完全独立：自带 PostgreSQL 容器，不同的容器名、网络和端口。

#### 部署架构

- **postgres** 服务（容器名 `llm-gateway-local-postgres`）
  - PostgreSQL 16，数据持久化到 Docker volume
  - 仅容器内部访问，不暴露端口到宿主机

- **backend** 服务（容器名 `llm-gateway-local-backend`）
  - 使用 `config.docker.local.yaml` 作为配置文件
  - 环境变量有默认值，零配置可启动

- **web** 服务（容器名 `llm-gateway-local-web`）
  - Nginx 反向代理，监听 3000 端口
  - 使用 `nginx.local.conf` 配置

#### 部署命令

```bash
# 零配置启动（使用内置默认值）
docker compose -f docker-compose.local.yml up --build -d

# 或使用自定义环境变量
cp .env.local.example .env.local
# 编辑 .env.local
docker compose -f docker-compose.local.yml --env-file .env.local up --build -d

# 查看服务状态
docker compose -f docker-compose.local.yml ps

# 查看日志
docker compose -f docker-compose.local.yml logs -f backend

# 停止（保留数据库数据）
docker compose -f docker-compose.local.yml down

# 停止并清除所有数据
docker compose -f docker-compose.local.yml down -v
```

#### 访问方式

- 前端：http://localhost:3000
- 健康检查：http://localhost:3000/health
- 首次注册使用 `ADMIN_INIT_TOKEN`（默认 `your-init-token`）成为管理员

#### 配置文件说明

- **`docker-compose.local.yml`** - 本地 Docker Compose 编排配置
- **`config.docker.local.yaml`** - 本地后端配置模板（使用环境变量占位符）
- **`.env.local.example`** - 环境变量示例文件
- **`.env.local`** - 本地环境变量（不提交到 git）

## 架构设计

### 核心组件

**代理层** (`internal/proxy/`, `internal/anthropic/`, `internal/responses/`, `internal/gemini/`)
- 处理多种格式的传入 API 请求
- 路由到合适的上游提供商
- 管理认证、计费和响应流式传输

**路由器** (`internal/router/`)
- 将模型名称映射到上游提供商池
- 支持模型别名（规范名称 vs 显示名称）
- 集成熔断器实现故障转移

**适配器** (`internal/adapter/`)
- 在 API 格式之间转换（OpenAI ↔ Anthropic）
- 实现协议无关的上游路由

**计费** (`internal/billing/`)
- 预扣费计费模型
- 通过工作池异步结算
- 支持余额、赠送金和佣金账户
- 基于 token 的定价，支持缓存定价

**存储** (`internal/store/`)
- PostgreSQL 数据访问层
- 管理用户、API 密钥、交易、模型、定价、订阅、租户

**配置** (`internal/config/`)
- 数据库驱动的模型和上游配置
- YAML 配置仅用于服务器设置
- 通过 `rebuildRouter()` 回调支持热重载

### API 端点

**公开代理端点：**
- `POST /v1/chat/completions` - OpenAI Chat Completions（Cursor、Windsurf、Aider）
- `POST /v1/messages` - Anthropic Messages（Codex）
- `POST /v1/responses` - OpenAI Responses（Codex CLI）
- `POST /gemini/v1/*` - Google Gemini 原生格式
- `GET /v1/models` - 列出可用模型（OpenAI 兼容）

**管理员端点**（需要 JWT + admin 角色）：
- `/api/admin/users` - 用户管理
- `/api/admin/pricing` - 定价配置
- `/api/admin/managed-keys` - 托管 API 密钥
- `/api/admin/dashboard` - 分析仪表板

**用户端点**（需要 JWT）：
- `/api/keys` - API 密钥管理
- `/api/billing/*` - 余额、交易、统计
- `/api/subscription/*` - 订阅计划和使用情况
- `/api/payment/*` - 支付宝支付操作

### 认证流程

1. 从 `Authorization` 头或 `x-api-key` 头提取 Bearer token
2. 使用 SHA256 哈希 token
3. 检查缓存 → 用户密钥 → 租户密钥 → 托管密钥
4. 验证密钥状态和余额
5. 将认证上下文附加到请求

### 计费流程

1. **预扣费**：根据 `max_tokens` 和模型定价估算最大成本
2. **代理请求**：转发到上游提供商，支持故障转移
3. **异步结算**：工作池处理响应中的实际使用量
4. **退还差额**：返还未使用的预扣费金额

### 模型配置

模型和上游配置**存储在数据库中**，而非 YAML 配置文件。使用管理后台可以：
- 添加/编辑/删除模型
- 配置上游提供商（base URL、API key、权重）
- 设置每个模型的定价（输入/输出/缓存 token）
- 测试上游连接性

通过管理 API 修改模型时，路由器会自动重建。

## 数据库迁移

迁移文件位于 `migrations/` 目录，通过 `golang-migrate` 在启动时自动执行。

```bash
# 启动时自动运行迁移
go run ./cmd/gateway -migrations ./migrations

# 手动迁移（如需要）
migrate -path migrations -database "postgres://..." up
```

迁移文件遵循命名模式：`{version}_{name}.up.sql` / `{version}_{name}.down.sql`

## 配置说明

**config.yaml**（仅服务器设置）：
- `server.port` - HTTP 监听端口（默认 9090）
- `server.no_pricing_strategy` - 如何处理无定价模型：`reject` / `warn` / `allow`
- `server.rate_limit.*` - 按 IP/用户/密钥限流
- `admin.jwt_secret` - JWT 签名密钥（生产环境必须修改）
- `admin.init_token` - 首个管理员用户注册所需的 token
- `database.dsn` - PostgreSQL 连接字符串
- `promotion.*` - 试用金、邀请奖励、首充赠送

**模型配置**通过管理后台管理，不在 YAML 中。

## 关键模式

### 同协议优先 + OpenAI Chat 兜底

上游通过 `protocols` 数组字段（多协议）声明支持的协议。每个入口先匹配同协议的上游做透传；该批上游全部失败时，回退到声明了 OpenAI Chat 兼容协议（`openai` / `openai-compatible`）的上游做协议转换兜底；都没有则返回 503 `no_compatible_upstream`。

- `/v1/messages` 与 `/v1/responses` 两个入口启用 OpenAI Chat 兜底
- `/v1/chat/completions` 与 `/gemini/...` 两个入口保持严格透传，不做协议转换
- 4xx 客户端错误不触发 fallback（由 `Failover` 内部 `isContextWindowExceededError` 等区分）

**例外**：`/v1/chat/completions` 入口通过 `responses.IsResponsesShapedBody(body)` 嗅探，将 Cursor Agent 错发的 Responses 形态请求转发到 `/v1/responses` handler。这是入口层面的转发，不是协议转换。

### 故障转移和熔断

每个上游都有一个熔断器（`internal/circuit/`）。负载均衡器（`internal/balancer/`）使用加权轮询分配负载。失败时，代理会尝试下一个上游。

### 流式响应

所有代理处理器都支持 SSE 流式传输：
- `internal/proxy/handler.go` - OpenAI Chat Completions 流式传输
- `internal/anthropic/handler.go` - Anthropic Messages 流式传输（直透传）
- `internal/responses/stream.go` - OpenAI Responses 流式传输

流式响应会被解析以提取 token 使用量用于计费。

### 订阅系统

支持带配额限制的分层订阅计划：
- 计划存储在 `subscription_plans` 表
- 用户订阅存储在 `user_subscriptions` 表
- 在计费层强制执行配额
- 升级/降级支持按比例退款

## 测试

运行测试：
```bash
go test ./...
```

关键测试文件：
- `internal/proxy/handler_test.go` - 代理处理器测试
- `internal/billing/service_test.go` - 计费逻辑测试
- `internal/anthropic/handler_test.go` - Anthropic API 测试
- `internal/responses/handler_test.go` - Responses API 测试

## 常见任务

### 添加新模型

1. 使用管理后台 → 模型 → 添加模型
2. 配置上游提供商（base URL、API key、权重）
3. 设置定价（输入/输出 token，如适用还包括缓存定价）
4. 路由器自动重建

### 调试计费问题

1. 检查 `transactions` 表中的预扣费和结算记录
2. 验证 `pricing` 表中该模型的定价是否存在
3. 检查 config.yaml 中的 `server.no_pricing_strategy`
4. 查看日志中的计费工作器错误

### 添加新 API 端点

1. 在适当的包中定义处理器（例如 `internal/admin/`）
2. 在 `cmd/gateway/main.go` 中注册路由
3. 如需认证则添加 JWT 中间件
4. 如仅限管理员则添加 admin 中间件

### 热重载模型配置

通过管理 API 修改模型会触发 `rebuildRouter()`：
1. 从数据库加载新配置
2. 使用更新的模型/上游构建新路由器
3. 在所有处理器中原子性地交换路由器
4. 无需重启服务器

## 前端开发

React 前端位于 `web/` 目录：
- `src/pages/` - 页面组件（Dashboard、Models、Billing 等）
- `src/components/` - 可复用 UI 组件（基于 shadcn/ui）
- `src/hooks/` - 自定义 React hooks（API 调用、SSE 等）
- `src/contexts/` - 认证上下文提供者
- `src/lib/` - 工具函数（API 客户端、查询键等）

Vite 开发服务器将 API 请求代理到 `localhost:9090` 的后端。

## 重要提示

- **永远不要提交密钥**：API 密钥、JWT 密钥、数据库密码应放在环境变量或 `.env` 文件中（已 gitignore）
- **数据库驱动配置**：模型和上游配置不在 YAML 配置中，使用管理后台
- **首个用户设置**：首个注册用户需要 `admin.init_token` 才能成为管理员
- **计费是异步的**：预扣费是同步的，结算通过工作池异步进行
- **熔断器**：上游在 `recovery_timeout` 后健康时自动恢复
- **限流**：按 IP、用户和 API 密钥独立配置
