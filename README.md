# RelayHub

> A universal notification delivery API — send via Discord, Email, SMS, and more through a single endpoint.

RelayHub is a self-hostable, multi-tenant notification platform. Instead of integrating each provider separately, you POST one request to RelayHub and it handles delivery, retries, fallback, and logging across channels.

---

## Phase 3 — Outbound Webhooks (Step 3 complete ✅)

| Feature | Status |
|---|---|
| `PUT /v1/webhook` — register a webhook URL (HTTPS only) | ✅ |
| Auto-generated `webhook_secret` (32-byte hex, shown once) | ✅ |
| Secret reused on URL update — explicit rotation not needed | ✅ |
| `DELETE /v1/webhook` — remove webhook configuration | ✅ |
| Push event on every final notification status (delivered / failed) | ✅ |
| HMAC-SHA256 signing — `X-RelayHub-Signature: sha256=<hex>` header | ✅ |
| Async fire — webhook never blocks `/v1/notify` response | ✅ |
| Retry with exponential backoff — up to 3 attempts, 5 s timeout each | ✅ |
| `webhook_deliveries` table — full audit log of every attempt | ✅ |
| `GET /v1/webhook/deliveries` — debug your webhook endpoint | ✅ |
| Works for both immediate and scheduled notifications | ✅ |

## Phase 3 — Scheduled Sends (Step 2 complete ✅)

| Feature | Status |
|---|---|
| `send_at` field on `POST /v1/notify` — queue for a future time | ✅ |
| 202 Accepted response with `status: "scheduled"` | ✅ |
| 30-day maximum schedule window (400 if exceeded) | ✅ |
| Background scheduler — polls every 30 s, claims atomically with `SKIP LOCKED` | ✅ |
| Multi-instance safe — no double-sends with concurrent app instances | ✅ |
| `GET /v1/notify/:request_id` — check delivery status | ✅ |
| `DELETE /v1/notify/:request_id` — cancel before it fires | ✅ |
| Graceful shutdown — scheduler stops cleanly on SIGTERM | ✅ |
| Templates + `send_at` together — variables resolved at request time | ✅ |
| Tenant isolation on all new endpoints | ✅ |

## Phase 3 — Templates (Step 1 complete ✅)

| Feature | Status |
|---|---|
| `POST /v1/templates` — create a reusable message template | ✅ |
| `GET /v1/templates` — list all templates for your tenant | ✅ |
| `GET /v1/templates/:name` — fetch a single template | ✅ |
| `PUT /v1/templates/:name` — update a template's body | ✅ |
| `DELETE /v1/templates/:name` — delete a template | ✅ |
| `POST /v1/notify` with `template` + `variables` — substitutes `{{placeholders}}` | ✅ |
| Missing-variable 400 — lists exactly which variables are absent | ✅ |
| Tenant isolation — templates are strictly scoped per tenant | ✅ |

## Phase 2 — Multi-tenancy (complete ✅)

| Feature | Status |
|---|---|
| `POST /v1/tenants` — register a new tenant account | ✅ |
| `X-API-Key` header authentication on all endpoints | ✅ |
| Per-tenant notification scoping (`tenant_id` on every log) | ✅ |
| `GET /v1/logs` only shows the authenticated tenant's data | ✅ |
| Rate limiting — 100 notifications/day on free plan | ✅ |
| `GET /v1/usage` — real-time usage stats from database | ✅ |

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

You can supply the message body in two ways — **plain message** or **template**. You must use exactly one; supplying both returns 400.

#### Option A — plain message

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

#### Option B — template + variables

Pass the name of a previously created template and a `variables` map. All `{{placeholders}}` in the template body are substituted before delivery.

```json
{
  "channel":   "email",
  "recipient": "ali@example.com",
  "template":  "order_shipped",
  "variables": {
    "customer_name": "Ali",
    "order_id":      "4471"
  }
}
```

#### Option C — scheduled send (`send_at`)

Add `"send_at"` (RFC3339) to any request shape. If the timestamp is in the future the notification is **queued, not sent**. The scheduler fires it within 30 seconds of the due time.

```json
{
  "channel":   "email",
  "recipient": "ali@example.com",
  "message":   "Your subscription renews tomorrow!",
  "send_at":   "2026-07-25T09:00:00Z"
}
```

You can combine templates + `send_at`:
```json
{
  "channel":   "email",
  "recipient": "ali@example.com",
  "template":  "order_shipped",
  "variables": { "customer_name": "Ali", "order_id": "4471" },
  "send_at":   "2026-07-25T09:00:00Z"
}
```
> Variables are resolved **at request time** and the final message text is stored. This prevents stale data if variables change before the scheduled time.

**Scheduled response (202 Accepted):**
```json
{
  "success": true,
  "data": {
    "request_id":    "550e8400-e29b-41d4-a716-446655440000",
    "status":        "scheduled",
    "scheduled_for": "2026-07-25T09:00:00Z"
  }
}
```

**Validation errors:**
- `send_at` not RFC3339 → 400
- `send_at` more than 30 days in the future → 400
- `send_at` in the past or omitted → sends immediately (existing behaviour)

**Missing variable response (400):**
If a placeholder in the template has no matching key in `variables`, the request is rejected and every missing key is listed:
```json
{
  "success":           false,
  "error":             "template is missing required variables",
  "missing_variables": ["order_id"]
}
```

**Both `message` and `template` provided (400):**
```json
{ "success": false, "error": "provide either 'message' or 'template', not both" }
```

**Success response (201):**
```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "status":     "delivered",
  "channel":    "email"
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

### Template API

All template endpoints require `X-API-Key`. Templates are strictly scoped to the authenticated tenant.

**Validation rules:**
- `name` — alphanumeric + underscores only, max 64 characters (`^[a-zA-Z0-9_]{1,64}$`)
- `body` — non-empty, max 4 000 characters

---

#### `POST /v1/templates`

Create a new reusable template.

```json
{ "name": "order_shipped", "body": "Hi {{customer_name}}, your order {{order_id}} has shipped!" }
```

**Success (201):**
```json
{
  "success": true,
  "data": {
    "id":         "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id":  "7a3f9c00-...",
    "name":       "order_shipped",
    "body":       "Hi {{customer_name}}, your order {{order_id}} has shipped!",
    "created_at": "2026-07-22T17:00:00Z",
    "updated_at": "2026-07-22T17:00:00Z"
  }
}
```

**Duplicate name (409):**
```json
{ "success": false, "error": "a template named \"order_shipped\" already exists" }
```

---

#### `GET /v1/templates`

List all templates for the authenticated tenant, sorted alphabetically by name.

```json
{
  "success": true,
  "data": {
    "count": 2,
    "templates": [
      { "id": "...", "name": "order_shipped", "body": "...", "created_at": "...", "updated_at": "..." },
      { "id": "...", "name": "welcome_email", "body": "...", "created_at": "...", "updated_at": "..." }
    ]
  }
}
```

---

#### `GET /v1/templates/:name`

Fetch a single template by name.

**Success (200):** returns the same shape as the single `data` object above.

**Not found (404):**
```json
{ "success": false, "error": "template not found: order_shipped" }
```

---

#### `PUT /v1/templates/:name`

Replace the body of an existing template.

```json
{ "body": "Hi {{customer_name}}, order {{order_id}} is on its way!" }
```

**Success (200):** returns the updated template object.
**Not found (404):** same shape as GET.

---

#### `DELETE /v1/templates/:name`

Permanently delete a template.

**Success (204):** empty body.
**Not found (404):** `{ "success": false, "error": "template not found: order_shipped" }`

---

### `GET /v1/notify/:request_id`

Returns the current status of any notification (immediate or scheduled). Requires `X-API-Key`.

Useful for polling a scheduled notification to see if it has fired yet.

**Success (200):**
```json
{
  "success": true,
  "data": {
    "id":            1,
    "tenant_id":     "...",
    "request_id":    "550e8400-e29b-41d4-a716-446655440000",
    "recipient":     "ali@example.com",
    "channel":       "email",
    "message":       "Hello Ali!",
    "status":        "scheduled",
    "scheduled_for": "2026-07-25T09:00:00Z",
    "created_at":    "2026-07-22T18:00:00Z"
  }
}
```

Possible `status` values: `scheduled`, `processing`, `delivered`, `failed`, `cancelled`.

**Not found (404):** returned for both non-existent IDs and IDs belonging to another tenant — existence is never leaked.

---

### `DELETE /v1/notify/:request_id`

Cancels a scheduled notification **before it fires**. Requires `X-API-Key`.

**Success (204):** empty body. The notification will never be sent.

**Already sent / in-progress (409):**
```json
{ "success": false, "error": "notification has already been sent or cancelled" }
```

**Not found (404):** as with GET, 404 covers both missing and wrong-tenant cases.

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

### Rate Limiting

The **free plan** allows **100 notifications per rolling 24-hour window** on `POST /v1/notify`.

**Rate limit response (429):**
```json
{
  "success":   false,
  "error":     "rate limit exceeded — free plan allows 100 notifications per 24-hour rolling window",
  "limit":     100,
  "remaining": 0,
  "resets_at": "2026-07-21T18:00:00Z"
}
```

Every allowed response also carries informational headers:
```
X-RateLimit-Limit:     100
X-RateLimit-Remaining: 88
X-RateLimit-Reset:     2026-07-21T18:00:00Z
```

> **Note (Phase 2 known simplification):** The in-memory rate-limit counter resets on process restart. The `GET /v1/usage` endpoint queries the database and is the source of truth for accurate counts. The in-memory limiter is the real-time enforcement layer for low-latency checking.

> Validation errors (400 Bad Request) do **not** consume a slot — only requests that reach the provider count against your limit.

---

### `GET /v1/usage`

Returns real-time usage statistics for the authenticated tenant, sourced directly from the database. Requires `X-API-Key`.

```json
{
  "success": true,
  "data": {
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "plan":      "free",
    "limit":     100,
    "used":      12,
    "remaining": 88,
    "resets_at": "2026-07-21T18:00:00Z"
  }
}
```

- `used` — notifications sent in the last 24 hours (from DB)
- `remaining` — slots left in the current window
- `resets_at` — when the oldest notification in the current window expires (i.e. when `used` will drop by 1)

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

# Check your usage stats (DB-backed, survives restarts)
curl -s http://localhost:8080/v1/usage \
  -H "X-API-Key: $API_KEY" | jq

# Health check (no auth needed)
curl -s http://localhost:8080/health

# Confirm 429 when over the rate limit (free plan: 100/day)
# X-RateLimit-* headers are present on every /v1/notify response
curl -s -i -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"recipient":"you@example.com","message":"test","channel":"email"}' | head -6

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
│   │   ├── auth.go                             # X-API-Key auth middleware + context helpers
│   │   └── ratelimit.go                        # In-memory rolling-window rate limiter
│   ├── providers/
│   │   ├── interface.go                        # Sender interface (the only contract core code touches)
│   │   ├── discord.go                          # Discord Webhook provider
│   │   └── email.go                            # Resend Email provider
│   ├── router/router.go                        # HTTP routes + handler methods
│   ├── service/notify_service/                 # NotifyService interface + Request/Response types
│   │   └── notify_service_impl/                # Retry, fallback, idempotency, DB logging
│   └── store/
│       ├── store.go                            # PostgreSQL store — tenants, notifications, templates, webhooks
│       └── template.go                         # SubstituteVars() — {{placeholder}} substitution engine
├── internal/scheduler/scheduler.go             # Background worker — claims and dispatches due notifications
├── internal/webhook/webhook.go                 # HMAC signing, event payload, async dispatcher
├── Dockerfile                                  # Multi-stage build
├── docker-compose.yml                          # App + Postgres
├── .env.example                                # Config template
└── README.md
```

---

## Outbound Webhooks

RelayHub can push a signed event to a URL you control whenever a notification reaches its final state (delivered or failed). This eliminates polling `GET /v1/notify/:id`.

### Configure Your Webhook

```http
PUT /v1/webhook
X-API-Key: <your-api-key>
Content-Type: application/json

{"webhook_url": "https://your-backend.com/relayhub-events"}
```

Response:
```json
{
  "success": true,
  "data": {
    "webhook_url": "https://your-backend.com/relayhub-events",
    "webhook_secret": "3a7f...c9d2"
  }
}
```

> ⚠️ **Save `webhook_secret` immediately.** It is only returned in full on this response. Subsequent `PUT` calls to update the URL will return the same secret — it is never regenerated unless you `DELETE` and re-register.

### Remove Your Webhook

```http
DELETE /v1/webhook
X-API-Key: <your-api-key>
```

Returns `204 No Content`.

### Event Payload

RelayHub POSTs the following JSON to your webhook URL:

```json
{
  "event": "notification.delivered",
  "request_id": "305d2c66-d3b5-4307-bf96-2338a0af0e28",
  "channel_used": "email",
  "fallback_used": false,
  "attempts": 1,
  "timestamp": "2026-07-24T17:14:00Z"
}
```

`event` is one of:
- `notification.delivered` — message reached the provider successfully
- `notification.failed` — all retry attempts exhausted

### Verifying the Signature

Every request includes an `X-RelayHub-Signature` header:

```
X-RelayHub-Signature: sha256=3a7fbc...
```

The signature is `HMAC-SHA256(webhook_secret_bytes, raw_request_body)` encoded as hex.

**Go verification snippet** (language-agnostic logic — see note below):

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "io"
    "net/http"
)

func verifyRelayHubSignature(r *http.Request, secret string) bool {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        return false
    }

    // Decode the hex secret into raw bytes.
    secretBytes, err := hex.DecodeString(secret)
    if err != nil {
        return false
    }

    // Compute expected HMAC-SHA256.
    mac := hmac.New(sha256.New, secretBytes)
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

    // Use constant-time comparison to prevent timing attacks.
    sig := r.Header.Get("X-RelayHub-Signature")
    return hmac.Equal([]byte(expected), []byte(sig))
}
```

> **Language-agnostic note:** The same logic applies in any language — compute `HMAC-SHA256` of the **raw request body bytes** using the hex-decoded secret, then compare the result (prefixed with `sha256=`) to the header value using a **constant-time** string comparison function.

### Inspect Delivery Attempts

```http
GET /v1/webhook/deliveries?limit=20
X-API-Key: <your-api-key>
```

Response:
```json
{
  "success": true,
  "data": {
    "count": 2,
    "deliveries": [
      {
        "id": 1,
        "tenant_id": "ca9a53bb-...",
        "notification_request_id": "305d2c66-...",
        "status_code": 200,
        "attempt": 1,
        "success": true,
        "created_at": "2026-07-24T17:14:31Z"
      }
    ]
  }
}
```

---

## Roadmap

- **Phase 1** ✅ Core engine — Discord provider, Email provider, delivery logs, retry, fallback, idempotency
- **Phase 2** ✅ Multi-tenancy — API key auth, per-tenant data isolation, rate limiting (100/day), usage stats
- **Phase 3 Step 1** ✅ Message templates — CRUD endpoints, `{{variable}}` substitution, tenant isolation
- **Phase 3 Step 2** ✅ Scheduled sends — `send_at`, background scheduler, cancel endpoint, multi-instance safe
- **Phase 3 Step 3** ✅ Outbound webhooks — HMAC-signed push events, async delivery, retry, audit log
- **Phase 4** 🔜 Redis Streams queue, worker pool, dead-letter queue
- **Phase 5** 🔜 React dashboard — logs, usage charts, template editor
