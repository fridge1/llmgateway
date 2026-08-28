[English](README.en.md) | [简体中文](README.md)

# LLM Gateway

LLM Gateway is a unified LLM API gateway and commercial platform. It consolidates OpenAI, Anthropic, and Google Gemini protocol endpoints behind a single authentication, routing, billing, subscription, tenant, and admin system — letting Claude Code, Codex CLI, Cursor, Windsurf, Aider, and other AI coding tools access all configured models through one gateway address and API key.

The project consists of a Go backend, a React web console, a Tauri desktop app, and a PostgreSQL database. Business data — models, upstreams, pricing, plans, tenants, orders — is database-driven, while YAML config only holds runtime and third-party integration parameters.

## Core Features

- **Multi-protocol LLM proxy**: Supports OpenAI Chat Completions, OpenAI Responses, Anthropic Messages, and native Gemini API, with conversion between OpenAI / Anthropic / Gemini protocols.
- **Routing & failover**: Model config lives in the database with support for model aliases, `gw/` prefixes, upstream protocol tagging, model overrides, weighted/priority routing, circuit breakers, and hot router reload.
- **Unified auth & access control**: User API keys, tenant API keys, and tenant sub-user API keys, with access gated by user status, plan scope, tenant permissions, and model permissions.
- **Billing & subscriptions**: Per-token, cached-token, or per-image billing with balance deduction, tenant balance, subscription quota, per-key usage stats, async settlement, and low-balance alerts.
- **Commercial operations**: Built-in Alipay top-up, orders, invoice titles & requests, recharge bonuses, referral rewards, check-ins, tasks, announcements, notifications, and audit logs.
- **Enterprise tenants**: Tenant management with member invites, role permissions, tenant-specific API keys, sub-user accounts, tenant ledgers, usage analytics, and custom pricing/discounts.
- **AI tool layer**: Built-in Playground, translation tool, image generation/editing tasks, image sharing, and a desktop app that configures local AI tools with one click.
- **Web & desktop consoles**: Web covers the marketing site, user console, admin dashboard, and org sub-account console; desktop covers login, balance, keys, usage, plans, tenants, and tool config.

## Tech Stack

| Layer | Tech |
|---|---|
| Backend | Go 1.26 · stdlib HTTP Server · golang-migrate |
| Database | PostgreSQL 16 |
| Web | React 19 · TypeScript · Vite · Tailwind CSS |
| Desktop | Tauri 2 · Rust · React · TypeScript |
| Payment | Alipay web/WAP payment · async notifications |
| SMS | Volcengine SMS |
| Object storage | Volcengine TOS |

## Quick Start

### Option 1: Local Docker Compose (recommended)

Local Compose bundles a PostgreSQL container — zero config for a full stack.

```bash
docker compose -f docker-compose.local.yml up --build -d
```

Access:

- Web console: http://localhost:3000
- Health check: http://localhost:3000/health
- First registration uses an init token (default `your-init-token`) to become admin

Common commands:

```bash
docker compose -f docker-compose.local.yml ps
docker compose -f docker-compose.local.yml logs -f backend
docker compose -f docker-compose.local.yml down      # stop (keep data)
docker compose -f docker-compose.local.yml down -v   # stop and wipe data
```

### Option 2: Source development

Prerequisites: Go ≥ 1.26, Node.js ≥ 18, PostgreSQL ≥ 15

```bash
# 1. Prepare config
cp config.example.yaml config.yaml
# Edit config.yaml: database.dsn, admin.jwt_secret, admin.init_token, etc.

# 2. Start backend (default port 9090)
go run ./cmd/gateway -config config.yaml -migrations ./migrations

# 3. Start frontend (port 5173, proxies to backend :9090)
cd web && npm install && npm run dev
```

### Option 3: Production Docker Compose

Production injects secrets via a `.env` file. See `.env.example` for the template.

```bash
cp .env.example .env   # fill in real secrets
docker compose up --build -d
```

Required environment variables:

- `JWT_SECRET`, `POSTGRES_PASSWORD`, `ADMIN_INIT_TOKEN`
- `SMS_ACCESS_KEY`, `SMS_SECRET_KEY`, `SMS_SIGN_NAME`, `SMS_ACCOUNT`, and SMS template IDs
- `ALIPAY_APP_ID`, `ALIPAY_NOTIFY_URL`, `ALIPAY_RETURN_URL`, `ALIPAY_IS_PRODUCTION`
- `TOS_*` object storage params (for image/PPT/invoice features)

## API Endpoints

### LLM-compatible endpoints

| Endpoint | Format | Typical use |
|---|---|---|
| `GET /v1/models` | OpenAI model list | Tool model discovery |
| `GET /v1/balance` | Gateway extension | Query balance by API key |
| `POST /v1/chat/completions` | OpenAI Chat Completions | Cursor, Windsurf, Aider, OpenAI SDK |
| `POST /v1/responses` | OpenAI Responses | Codex CLI, Responses API clients |
| `POST /v1/messages` | Anthropic Messages | Claude Code, Anthropic SDK |
| `POST /v1/messages/count_tokens` | Anthropic token counting | Claude-compatible tools |
| `POST /gemini/v1/models/{model}:generateContent` | Native Gemini | Gemini SDK |
| `POST /gemini/v1beta/models/{model}:streamGenerateContent` | Gemini SSE | Gemini streaming |
| `POST /v1/images/generations` | OpenAI image generation | Sync image generation |
| `POST /v1/images/edits` | OpenAI image edit | Sync image editing |
| `POST /v1/images/tasks` | Gateway async image task | Task-based generation/polling |

Auth via `Authorization: Bearer <API_KEY>`; the Anthropic Messages endpoint also accepts `x-api-key: <API_KEY>`.

### Console API

- **Auth**: `/api/login`, `/api/register`, `/api/sms/send`, `/api/reset-password`
- **User**: `/api/keys`, `/api/billing/*`, `/api/subscription/*`, `/api/payment/*`
- **Models**: `/api/models`, `/api/pricing`
- **Tenants**: `/api/tenants/*`, `/api/sub-user/*`
- **Tools**: `/api/image/*`, `/api/ppt/*`, `/api/chat/*`, `/api/invoice/*`
- **Admin**: `/api/admin/*` (models, upstreams, pricing, users, orders, plans, tenants, invoices, audit)

## Configuration

Business config — models, upstreams, pricing — is stored in the database and managed via the admin console. Changes trigger an automatic router rebuild with no restart. `config.yaml` only holds runtime settings:

| Setting | Description |
|---|---|
| `server.port` | Backend listen port (default `9090`) |
| `server.request_timeout` | Upstream request timeout |
| `server.max_request_body_bytes` | Request body size limit |
| `server.no_pricing_strategy` | Unpriced model policy: `reject` / `warn` / `allow` |
| `server.rate_limit.*` | Rate limiting by IP, user, and key |
| `admin.jwt_secret` | JWT signing secret — must change in production |
| `admin.init_token` | Token required for first admin registration |
| `database.dsn` | PostgreSQL connection string (env var expansion supported) |
| `payment.alipay.*` | Alipay configuration |
| `sms.*` | SMS service configuration |
| `tos.*` | Object storage configuration |
| `promotion.*` | Signup bonus, recharge bonus, referral rewards |
| `retention.*` | Low-balance alerts, check-in rewards, retention policies |

## Project Structure

```text
cmd/gateway/                 # Main service entry & HTTP routes
cmd/compensate-referral/     # Referral reward compensation tool
internal/proxy/              # OpenAI Chat proxy core, auth, failover, streaming
internal/anthropic/          # Anthropic Messages compatibility layer
internal/responses/          # OpenAI Responses compatibility layer
internal/gemini/             # Native Gemini API compatibility layer
internal/adapter/            # Cross-protocol request conversion
internal/router/            # Model-to-upstream routing table
internal/balancer/          # Upstream load balancing
internal/circuit/           # Circuit breakers
internal/billing/           # Billing, subscription usage, settlement
internal/pricing/           # Model pricing cache & management
internal/subscription/      # Plans, quota, subscription orders
internal/payment/           # Alipay payment & async notifications
internal/store/             # PostgreSQL data access layer
internal/admin/             # Admin dashboard APIs
internal/image/             # Image generation/editing
internal/ppt/               # PPT generation tasks
internal/invoice/           # Invoice management
migrations/                 # SQL migration files
web/                        # React web app & Nginx config
desktop/                    # Tauri desktop app
scripts/                    # Ops & data repair scripts
```

## Development & Testing

```bash
# Backend tests
go test ./...

# Web build & lint
cd web && npm run build && npm run lint

# Desktop build
cd desktop && npm run tauri build

# Health check
curl http://localhost:9090/health
curl http://localhost:9090/v1/models
```

## Security Notes

- Never commit `.env`, real API keys, JWT secrets, database passwords, Alipay private keys, or storage credentials.
- Production must replace `admin.jwt_secret` and set a non-default `admin.init_token`.
- Alipay async notifications require a public HTTPS URL matching `notify_url` in the Alipay dashboard.
- Image, PPT, and invoice features depend on object storage; they degrade or are unavailable without TOS config.

## License

Licensed under the [PolyForm Noncommercial License 1.0.0](LICENSE).

For personal study, research, and internal non-commercial use only. **Commercial use is prohibited**, including but not limited to selling, paid hosting, and SaaS resale. See the [LICENSE](LICENSE) file for details.
