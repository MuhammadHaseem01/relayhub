# RelayHub

> A universal notification delivery API — send via Discord, Email, SMS, and more through a single endpoint.

RelayHub is a self-hostable, multi-tenant notification platform. Instead of integrating each provider separately, you POST one request to RelayHub and it handles delivery, retries, fallback, and logging across channels.

---

## Phase 1 — What's implemented

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
- Your Telegram bot token from Step 1 above

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

### `POST /v1/notify`

Send a notification.

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

**Success response (200):**
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

Returns recent delivery attempts, newest first.

```json
{
  "count": 2,
  "logs": [
    {
      "id":            1,
      "request_id":    "550e8400-...",
      "recipient":     "987654321",
      "channel":       "discord",
      "message":       "Hello!",
      "status":        "delivered",
      "error_message": "",
      "attempts":      1,
      "fallback_used": false,
      "created_at":    "2026-07-12T18:00:00Z"
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
# Send a Discord message
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN",
    "message":   "Hello from RelayHub! 🚀",
    "channel":   "discord"
  }' | jq

# Send an Email message
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "you@example.com",
    "message":   "Hello from RelayHub via Email! 🚀",
    "channel":   "email"
  }' | jq

# Use Auto-Fallback (Discord -> Email)
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -d '{
    "discord_recipient": "https://discord.com/api/webhooks/INVALID_ID/INVALID_TOKEN",
    "email_recipient":   "you@example.com",
    "message":           "Fallback test message!",
    "channel":           "auto"
  }' | jq

# Send an Idempotent request (prevents duplicate sends)
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: my-unique-key-123" \
  -d '{
    "recipient": "https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN",
    "message":   "Hello exactly once! 🚀",
    "channel":   "discord"
  }' | jq

# View delivery logs
curl -s http://localhost:8080/v1/logs | jq

# Health check
curl -s http://localhost:8080/health
```

---

## Project structure

```
relayhub/
├── cmd/server/main.go              # Entrypoint — wires config, DB, providers, router
├── internal/
│   ├── config/config.go           # Environment variable loader
│   ├── providers/
│   │   ├── interface.go           # Sender interface (the only contract core code touches)
│   │   ├── discord.go             # Discord Webhook provider
│   │   └── email.go               # Resend Email provider
│   ├── handlers/notify.go         # POST /notify + GET /logs HTTP handlers
│   └── store/store.go             # PostgreSQL store + auto-migration
├── Dockerfile                     # Multi-stage build
├── docker-compose.yml             # App + Postgres
├── .env.example                   # Config template
└── README.md
```

---

## Roadmap

- **Phase 1** ✅ Core engine — Discord provider, Email provider, delivery logs, retry, fallback, idempotency
- **Phase 2** 🔜 Multi-tenancy — API keys, per-tenant rate limiting
- **Phase 3** 🔜 Templates, scheduled sends, outbound webhooks, Discord + SMTP
- **Phase 4** 🔜 Redis Streams queue, worker pool, dead-letter queue
- **Phase 5** 🔜 React dashboard — logs, usage charts, template editor
