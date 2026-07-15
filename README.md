# RelayHub

> A universal notification delivery API — send via Telegram, Email, SMS, and more through a single endpoint.

RelayHub is a self-hostable, multi-tenant notification platform. Instead of integrating each provider separately, you POST one request to RelayHub and it handles delivery, retries, fallback, and logging across channels.

---

## Phase 1 — What's implemented

| Feature | Status |
|---|---|
| `POST /v1/notify` — send a Telegram message | ✅ |
| `POST /v1/notify` — send an Email with fallback | ✅ |
| `GET /v1/logs` — view recent delivery history | ✅ |
| `GET /health` — health check endpoint | ✅ |
| Delivery log persisted in PostgreSQL | ✅ |
| Structured JSON logging with `request_id` | ✅ |
| Pluggable provider interface (`Sender`) | ✅ |
| Docker Compose (app + postgres) | ✅ |

---

## Getting a free Telegram Bot token

You need two things: a **bot token** and your **chat_id**.

### Step 1 — Create a bot and get the token

1. Open Telegram and search for **@BotFather**
2. Send `/newbot`
3. Follow the prompts — choose a name and username for your bot
4. BotFather will reply with your token, looking like:
   ```
   1234567890:AAFxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   ```
5. Copy it — this is your `TELEGRAM_BOT_TOKEN`

### Step 2 — Get your chat_id

1. Start a conversation with your new bot (search for it in Telegram and click **Start**)
2. Send any message to the bot (e.g. "hello")
3. In your browser, open:
   ```
   https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates
   ```
4. Look for `"chat":{"id":XXXXXXX}` in the response — that number is your **chat_id**

> **Tip:** Your chat_id is typically a large positive integer for personal chats (e.g. `987654321`).
> For group chats it's a negative integer (e.g. `-1001234567890`).

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

# 3. Fill in your bot token and Resend key in .env
#    Open .env and replace:
#    TELEGRAM_BOT_TOKEN="your_token"
#    RESEND_API_KEY="re_your_key"
#    FROM_EMAIL="onboarding@resend.dev"

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

**Request body (Telegram or Email):**
```json
{
  "recipient": "987654321",
  "message":   "Hello from RelayHub! 🚀",
  "channel":   "telegram" // or "email"
}
```

**Request body (Auto Fallback):**
If channel is "auto", the system will try Telegram first. If it completely fails, it will automatically fall back to Email.
```json
{
  "message":            "Hello from RelayHub! 🚀",
  "channel":            "auto",
  "telegram_recipient": "987654321",
  "email_recipient":    "you@example.com"
}
```

**Success response (200):**
```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "status":     "delivered",
  "channel":    "telegram"
}
```

**Failure response (502):**
```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "status":     "failed",
  "channel":    "telegram",
  "error":      "telegram: API error 400: chat not found"
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
      "channel":       "telegram",
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
# Send a Telegram message
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "YOUR_CHAT_ID",
    "message":   "Hello from RelayHub! 🚀",
    "channel":   "telegram"
  }' | jq

# Send an Email message
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "you@example.com",
    "message":   "Hello from RelayHub via Email! 🚀",
    "channel":   "email"
  }' | jq

# Use Auto-Fallback (Telegram -> Email)
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -d '{
    "telegram_recipient": "INVALID_CHAT_ID_TO_FORCE_FALLBACK",
    "email_recipient":    "you@example.com",
    "message":            "Fallback test message!",
    "channel":            "auto"
  }' | jq

# Send an Idempotent request (prevents duplicate sends)
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: my-unique-key-123" \
  -d '{
    "recipient": "987654321",
    "message":   "Hello exactly once! 🚀",
    "channel":   "telegram"
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
│   │   └── telegram.go            # Telegram Bot API provider
│   ├── handlers/notify.go         # POST /notify + GET /logs HTTP handlers
│   └── store/store.go             # PostgreSQL store + auto-migration
├── Dockerfile                     # Multi-stage build
├── docker-compose.yml             # App + Postgres
├── .env.example                   # Config template
└── README.md
```

---

## Roadmap

- **Phase 1** ✅ Core engine — Telegram provider, delivery logs
- **Phase 2** 🔜 Multi-tenancy — API keys, per-tenant rate limiting
- **Phase 3** 🔜 Templates, scheduled sends, outbound webhooks, Discord + SMTP
- **Phase 4** 🔜 Redis Streams queue, worker pool, dead-letter queue
- **Phase 5** 🔜 React dashboard — logs, usage charts, template editor
