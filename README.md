# RelayHub

> A universal notification delivery API — send via Discord, Email, SMS, and more through a single endpoint.

RelayHub is a self-hostable, multi-tenant notification platform. Instead of integrating each provider separately, you POST one request to RelayHub and it handles delivery, retries, fallback, and logging across channels.

---

## Phase 2 — Multi-tenancy (current)

| Feature | Status |
|---|---|
| `POST /v1/tenants` — register a new tenant account | ✅ |
| `X-API-Key` header authentication on all endpoints | ✅ |
| Per-tenant notification scoping (`tenant_id` on every log) | ✅ |
| `GET /v1/logs` only shows the authenticated tenant's data | ✅ |

## Phase 1 — Core engine

| Feature | Status |
|---|---|
| `POST /v1/notify` — send a Discord message | ✅ |
| `POST /v1/notify` — send an Email with fallback | ✅ |
| `GET /v1/logs` — view recent delivery history | ✅ |
| `GET /health` — health check endpoint | ✅ |
| Delivery log persisted in PostgreSQL | ✅ |
| Structured JSON logging with `request_id` | ✅ |
| Pluggable provider interface (`Sender`) | ✅ |
| Docker Compose (app + postgres) | ✅ |

---

## Quick Start

```bash
# 1. Clone the repo
git clone https://github.com/yourusername/relayhub.git
cd relayhub

# 2. Create your .env from the template
cp .env.example .env

# 3. Fill in your API keys in .env

# 4. Start everything
docker compose up --build
```

Then register a tenant and grab your API key:

```bash
curl -s -X POST http://localhost:8080/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "My App"}' | jq
# → { "data": { "tenant_id": "...", "api_key": "rh_..." } }
```

---

## Getting a free Discord Webhook URL

Discord webhooks are free, require no bot token, and work without any approval process.

### Step 1 — Create (or open) a Discord server

1. Open Discord and click the **+** icon on the left sidebar
2. Choose **Create My Own** → **For me and my friends** → give it any name

### Step 2 — Create a Webhook on a channel

1. Right-click any text channel (e.g. `#general`) → **Edit Channel**
2. Go to **Integrations** → **Webhooks** → **New Webhook**
3. Give it a name (e.g. "RelayHub"), then click **Copy Webhook URL**
4. The URL will look like:
   ```
   https://discord.com/api/webhooks/1234567890/abcdefghijklmnopqrstuvwxyz
   ```
5. Paste this as your `recipient` in API requests, or set it as `DISCORD_WEBHOOK_URL` in `.env` to use as a default.

> **Tip:** You can create multiple webhooks on different channels and route different notifications to each one by passing the webhook URL as `recipient` at request time.

---

## Getting a free Resend API key (Email Provider)

You can send up to 3,000 emails per month for free using Resend.

1. Go to [resend.com](https://resend.com) and create a free account (no credit card required).
2. Once logged in, go to **API Keys** on the sidebar.
3. Click **Create API Key**. Give it a name and ensure it has "Sending access".
4. Copy the generated key. It will look like: `re_123456789...`
5. This is your `RESEND_API_KEY`.
6. For the `FROM_EMAIL`, you can use `onboarding@resend.dev` to test sending emails to the address you signed up with.

---

## Running locally with Docker Compose

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose installed

### Setup

```bash
# 1. Clone the repo
git clone https://github.com/yourusername/relayhub.git
cd relayhub

# 2. Create your .env from the template
cp .env.example .env

# 3. Fill in your Resend key in .env
#    Open .env and replace:
#    RESEND_API_KEY="re_your_key"
#    FROM_EMAIL="onboarding@resend.dev"
#    DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..." (optional default)

# 4. Start everything
docker compose up --build
```

The API will be available at **http://localhost:8080**.

### Stopping

```bash
docker compose down          # stop containers, keep DB data
docker compose down -v       # stop and delete DB data
```

---

## API Reference

### `POST /v1/tenants` *(no authentication required)*

Register a new tenant account. This is how you sign up — you cannot use any other endpoint until you have an API key.

**Request body:**
```json
{ "name": "My Application" }
```

**Success response (201):**
```json
{
  "success": true,
  "data": {
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "api_key":   "rh_a3f9c2d1e4b7f8a2c5d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
  }
}
```

> **Keep your `api_key` secret.** Pass it in the `X-API-Key` header on every subsequent request.

---

### Authentication

All endpoints except `POST /v1/tenants` and `GET /health` require a valid API key:

```
X-API-Key: rh_your_api_key_here
```

**Missing key response (401):**
```json
{ "success": false, "error": "X-API-Key header is required" }
```

**Invalid key response (401):**
```json
{ "success": false, "error": "invalid API key" }
```

---

### `POST /v1/notify`

Send a notification. Requires `X-API-Key`.

**Request body (Discord or Email):**
```json
{
  "recipient": "https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN",
  "message":   "Hello from RelayHub! 🚀",
  "channel":   "discord"
}
```

> **Note:** For `channel=discord`, `recipient` is the full Discord Webhook URL.
> For `channel=email`, `recipient` is an email address.

**Request body (Auto Fallback):**
If channel is `"auto"`, the system will try Discord first (with retries). If it completely fails, it automatically falls back to Email.
```json
{
  "message":          "Hello from RelayHub! 🚀",
  "channel":          "auto",
  "discord_recipient": "https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN",
  "email_recipient":  "you@example.com"
}
```

**Success response (201):**
```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "status":     "delivered",
  "channel":    "discord"
}
```

**Failure response (502):**
```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "status":     "failed",
  "channel":    "discord",
  "error":      "discord: webhook not found (404) — check the webhook URL"
}
```

---

### `GET /v1/logs?limit=50`

Returns your tenant's recent delivery attempts, newest first. Requires `X-API-Key`.

> Logs are **strictly scoped to your tenant** — you can never see another tenant's data.

```json
{
  "count": 2,
  "logs": [
    {
      "id":            1,
      "tenant_id":     "550e8400-e29b-41d4-a716-446655440000",
      "request_id":    "550e8400-...",
      "recipient":     "you@example.com",
      "channel":       "email",
      "message":       "Hello!",
      "status":        "delivered",
      "error_message": "",
      "attempts":      1,
      "fallback_used": false,
      "created_at":    "2026-07-20T18:00:00Z"
    }
  ]
}
```

---

### `GET /health`

```json
{ "status": "ok", "service": "relayhub" }
```

---

## Example curl commands

```bash
# Step 1 — Register a tenant (do this once)
curl -s -X POST http://localhost:8080/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "My App"}' | jq
# Save the api_key from the response

export API_KEY="rh_your_key_here"

# Step 2 — Send a Discord message
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "recipient": "https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN",
    "message":   "Hello from RelayHub! 🚀",
    "channel":   "discord"
  }' | jq

# Send an Email message
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "recipient": "you@example.com",
    "message":   "Hello from RelayHub via Email! 🚀",
    "channel":   "email"
  }' | jq

# Use Auto-Fallback (Discord → Email)
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "discord_recipient": "https://discord.com/api/webhooks/INVALID_ID/INVALID_TOKEN",
    "email_recipient":   "you@example.com",
    "message":           "Fallback test message!",
    "channel":           "auto"
  }' | jq

# Send an Idempotent request (prevents duplicate sends)
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -H "X-Idempotency-Key: my-unique-key-123" \
  -d '{
    "recipient": "https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN",
    "message":   "Hello exactly once! 🚀",
    "channel":   "discord"
  }' | jq

# View your delivery logs (only your tenant's data)
curl -s http://localhost:8080/v1/logs \
  -H "X-API-Key: $API_KEY" | jq

# Health check (no auth needed)
curl -s http://localhost:8080/health

# Confirm 401 for missing key
curl -s http://localhost:8080/v1/logs | jq

# Confirm 401 for invalid key
curl -s http://localhost:8080/v1/logs \
  -H "X-API-Key: rh_wrong" | jq
```

---

## Project structure

```
relayhub/
├── cmd/server/main.go                          # Entrypoint — wires config, DB, providers, router
├── internal/
│   ├── config/config.go                        # Environment variable loader
│   ├── middleware/
│   │   └── auth.go                             # X-API-Key auth middleware + context helpers
│   ├── providers/
│   │   ├── interface.go                        # Sender interface (the only contract core code touches)
│   │   ├── discord.go                          # Discord Webhook provider
│   │   └── email.go                            # Resend Email provider
│   ├── router/router.go                        # HTTP routes + handler methods
│   ├── service/notify_service/                 # NotifyService interface + Request/Response types
│   │   └── notify_service_impl/                # Retry, fallback, idempotency, DB logging
│   └── store/store.go                          # PostgreSQL store — tenants + notifications
├── Dockerfile                                  # Multi-stage build
├── docker-compose.yml                          # App + Postgres
├── .env.example                                # Config template
└── README.md
```

---

## Roadmap

- **Phase 1** ✅ Core engine — Discord provider, Email provider, delivery logs, retry, fallback, idempotency
- **Phase 2** ✅ Multi-tenancy — API key auth, per-tenant data isolation
- **Phase 3** 🔜 Rate limiting, usage stats, and quotas per plan
- **Phase 4** 🔜 Templates, scheduled sends, outbound webhooks, Discord + SMTP
- **Phase 5** 🔜 Redis Streams queue, worker pool, dead-letter queue
- **Phase 6** 🔜 React dashboard — logs, usage charts, template editor
