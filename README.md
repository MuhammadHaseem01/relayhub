# RelayHub

> A universal notification delivery API — send via Discord, Email, SMS, and more through a single endpoint.

RelayHub is a self-hostable, multi-tenant notification platform. Instead of integrating each provider separately, you POST one request to RelayHub and it handles delivery, retries, fallback, and logging across channels.

---

## Phase 5 — Full React Dashboard (Step 1 & Step 2 complete ✅)

| Feature | Status |
|---|---|
| React + Vite + TypeScript frontend application in `/web` directory | ✅ |
| Custom dark engineering theme (`Space Grotesk`, `Inter`, `JetBrains Mono` + Lucide vector icons) | ✅ |
| Tenant Auth Flow — Register new tenant (shows API key once with copy) or paste existing key | ✅ |
| API client module (`/web/src/api/client.ts`) with automatic `X-API-Key` header handling | ✅ |
| CORS middleware on Go backend allowing dev origin (`http://localhost:5173`) | ✅ |
| **Dashboard Overview Page** — welcome header, 4 stat cards (Total Sent, Success Rate, Active Providers, Rate Limit), recent activity table | ✅ |
| **Send Notification Page** — testing form for `email`, `discord`, `smtp`, `auto`, idempotency, `send_at`, live inspector, JSON panel | ✅ |
| **Delivery Logs Page** — auto-refreshing table (5s interval countdown) of tenant delivery history | ✅ |
| **Templates Page** — template CRUD, client-side live Handlebars `{{variables}}` preview | ✅ |
| **Webhooks Page** — HTTPS endpoint manager, one-time HMAC secret reveal, recent webhook deliveries, signature verification guide | ✅ |
| **Dead Letter Page** — DLQ listing, positive empty state, one-click job replay | ✅ |
| **Provider Health Page** — circuit breaker status grid (discord, email, smtp), 5s auto-refresh, open circuit warning banner | ✅ |
| **Usage Page** — 24-hour quota usage, color-shifting progress bar (accent/warning/danger), relative reset timer | ✅ |

## Phase 4 — DLQ & Provider Health Circuit Breaker (Step 2 complete ✅)

| Feature | Status |
|---|---|
| Dead-Letter Queue stream (`relayhub:notifications:dlq`) & `status: "dead_letter"` DB state | ✅ |
| Automatic DLQ promotion after 3 failed worker-level attempts | ✅ |
| `GET /v1/notify/dead-letter` — list dead-lettered notifications for tenant | ✅ |
| `POST /v1/notify/:request_id/replay` — re-enqueue dead-lettered notification for retry | ✅ |
| In-memory circuit breaker per provider (5 consecutive failures → 60s open window) | ✅ |
| Automatic circuit bypass in `channel="auto"` (skips open-circuit provider immediately) | ✅ |
| Half-open trial recovery (single request probe after 60s window) | ✅ |
| `GET /v1/health/providers` — public monitoring endpoint for provider health states | ✅ |

## Phase 4 — Async Queue & Background Workers (Step 1 complete ✅)

| Feature | Status |
|---|---|
| `POST /v1/notify` async processing — returns instant `201 Created` with `status: "queued"` | ✅ |
| Redis Streams queue engine (`relayhub:notifications` stream, `relayhub-workers` group) | ✅ |
| `WORKER_COUNT` configurable background worker pool (default: 5 goroutines) | ✅ |
| XREADGROUP consumer group processing with automatic XACK after delivery write | ✅ |
| Periodic worker crash recovery via `XAUTOCLAIM` (reclaims jobs idle > 90 s every 60 s) | ✅ |
| Shared delivery engine (`dispatch.Executor.Run`) — zero code duplication across workers and scheduler | ✅ |
| Status query (`GET /v1/notify/:request_id`) — track delivery progress from `queued` to `delivered`/`failed` | ✅ |
| Seamless docker-compose integration with `redis:7-alpine` | ✅ |

## Phase 3 — SMTP Provider (Step 4 complete ✅)

| Feature | Status |
|---|---|
| `channel=smtp` — second independent email path via plain SMTP | ✅ |
| Stdlib-only: `net/smtp` + `crypto/tls`, zero new dependencies | ✅ |
| STARTTLS (port 587) and implicit TLS/SMTPS (port 465) | ✅ |
| Unauthenticated mode for local catchers (Mailpit, MailHog) | ✅ |
| Clear errors: auth failure, connection refused, invalid recipient | ✅ |
| Config via `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | ✅ |
| Unit tests — in-process mock SMTP server, no real credentials needed | ✅ |
| `channel=email` (Resend) unchanged — zero breaking change | ✅ |
| Adapter pattern proven: zero changes to core service / retry logic | ✅ |

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

## Getting free SMTP credentials (SMTP Provider)

RelayHub's `channel=smtp` uses plain SMTP — any server that speaks the protocol works.
Two free options are covered below; **Mailtrap** is recommended because you can see captured messages in a nice web UI without anything landing in a real inbox.

### Option A — Mailtrap sandbox *(recommended for testing)*

[Mailtrap](https://mailtrap.io) captures outgoing emails in a sandboxed inbox — nothing is ever delivered to real recipients, which makes it perfect for development.

1. Go to [mailtrap.io](https://mailtrap.io) and sign up for a free account (no credit card).
2. In the dashboard, go to **Email Testing → Inboxes**.
3. Click the default inbox (or create a new one).
4. Select **SMTP** from the integration dropdown. You will see credentials like:
   ```
   Host:     sandbox.smtp.mailtrap.io
   Port:     587
   Username: <your-mailtrap-username>
   Password: <your-mailtrap-password>
   ```
5. Copy these four values into your `.env`:
   ```env
   SMTP_HOST=sandbox.smtp.mailtrap.io
   SMTP_PORT=587
   SMTP_USERNAME=<your-mailtrap-username>
   SMTP_PASSWORD=<your-mailtrap-password>
   SMTP_FROM=test@myapp.dev
   ```
6. Send a test notification (see curl example below) and watch the email appear in Mailtrap's inbox.

> **Free tier:** 1,000 emails/month, unlimited inboxes, full API access — no card required.

### Option B — Gmail App Password *(sends real emails)*

If you need emails to actually land in an inbox, use a Gmail account with an App Password.
This requires 2-Step Verification to be enabled on your Google Account.

1. Go to [myaccount.google.com/security](https://myaccount.google.com/security) and ensure **2-Step Verification** is on.
2. Visit [myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords).
3. Under "Select app" choose **Mail**; under "Select device" choose **Other** and type `RelayHub`.
4. Click **Generate** — Google shows a 16-character password (e.g. `abcd efgh ijkl mnop`). Copy it (spaces are ignored).
5. Add to your `.env`:
   ```env
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USERNAME=you@gmail.com
   SMTP_PASSWORD=abcdefghijklmnop   # the 16-char app password, no spaces
   SMTP_FROM=you@gmail.com
   ```

> **Note:** Gmail App Passwords send real emails and count toward your Gmail sending limits. Use Mailtrap for load testing or when you don't want to risk landing in spam.

### curl example — send via SMTP

```bash
curl -s -X POST http://localhost:8080/v1/notify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: rh_your_api_key" \
  -d '{
    "channel":   "smtp",
    "recipient": "you@example.com",
    "message":   "Hello from RelayHub via SMTP!"
  }' | jq
```

Expected response:
```json
{
  "success": true,
  "data": {
    "request_id": "550e8400-...",
    "channel":    "smtp",
    "status":     "delivered"
  }
}
```

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

**Channel values:**

| `channel` | Provider | `recipient` format |
|---|---|---|
| `discord` | Discord Webhook | Full Discord Webhook URL |
| `email` | Resend API | Email address |
| `smtp` | Plain SMTP server | Email address |
| `auto` | Discord → Resend fallback | Needs `discord_recipient` + `email_recipient` |

**Request body (Discord):**
```json
{
  "recipient": "https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN",
  "message":   "Hello from RelayHub! 🚀",
  "channel":   "discord"
}
```

**Request body (Email via Resend):**
```json
{
  "recipient": "you@example.com",
  "message":   "Hello from RelayHub! 🚀",
  "channel":   "email"
}
```

**Request body (Email via SMTP):**
```json
{
  "recipient": "you@example.com",
  "message":   "Hello from RelayHub! 🚀",
  "channel":   "smtp"
}
```

**Request body (Auto Fallback — Discord then Resend):**
If channel is `"auto"`, the system will try Discord first (with retries). If it completely fails, it automatically falls back to Resend email.
```json
{
  "message":           "Hello from RelayHub! 🚀",
  "channel":           "auto",
  "discord_recipient": "https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN",
  "email_recipient":   "you@example.com"
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

**Success response (201 Created):**
Notifications are validated (auth, rate limits, template rendering, idempotency) and written to Postgres with `status: "queued"`, then enqueued into Redis Streams for background worker pickup. The API returns **instantly** without waiting for network provider delivery:
```json
{
  "success": true,
  "data": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "status":     "queued"
  }
}
```

#### Async Delivery & Worker Pool

1. **Queueing:** `POST /v1/notify` validates the request, writes a `status = 'queued'` notification record, pushes a job to Redis Stream `relayhub:notifications`, and returns `201 Created` immediately.
2. **Worker Processing:** Background worker goroutines (`WORKER_COUNT`, default 5) consume jobs from consumer group `relayhub-workers` using `XREADGROUP`.
3. **Execution:** Workers run the delivery attempt via `dispatch.Executor` (handles retries, exponential back-off, channel fallbacks, Postgres status updates, and outbound webhooks), then issue `XACK`.
4. **Crash Recovery:** A background reclaimer process runs every 60 seconds using `XAUTOCLAIM` to re-assign any unacknowledged jobs idle for >90 seconds (protecting against worker crashes).
5. **Checking Final Status:** Call `GET /v1/notify/:request_id` to inspect the final delivery state (`delivered`, `failed`, or `dead_letter`).

---

### `GET /v1/health/providers` *(no authentication required)*

Public monitoring endpoint for inspecting the real-time circuit breaker health of all underlying providers (`discord`, `email`, `smtp`).

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "discord": "healthy",
    "email":   "unhealthy",
    "smtp":    "healthy"
  }
}
```

---

### `GET /v1/notify/dead-letter`

List notifications that have permanently failed after 3 worker-level attempts and landed in the Dead-Letter Queue (`relayhub:notifications:dlq`). Requires `X-API-Key`. Scoped to the authenticated tenant.

**Query parameters:**
- `limit` (optional): maximum records to return (default 50, max 200).

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "count": 1,
    "notifications": [
      {
        "id": 42,
        "tenant_id": "550e8400-...",
        "request_id": "9b1deb4d-...",
        "recipient": "invalid-address@example.com",
        "channel": "email",
        "message": "Hello!",
        "status": "dead_letter",
        "error_message": "exceeded maximum worker attempts",
        "attempts": 3,
        "created_at": "2026-07-29T16:00:00Z"
      }
    ]
  }
}
```

---

### `POST /v1/notify/:request_id/replay`

Re-enqueue a dead-lettered notification back onto the main Redis stream for another delivery attempt. Resets worker attempt count to 0 and updates Postgres status to `queued`. Requires `X-API-Key`.

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "request_id": "9b1deb4d-...",
    "status":     "queued"
  }
}
```

**Errors:**
- `404 Not Found`: notification does not exist or belongs to another tenant.
- `409 Conflict`: notification is not currently in `dead_letter` status.

---

## Reliability & Failure Handling Architecture

RelayHub is designed to handle failure at scale without losing notifications or blocking callers:

```
[ POST /v1/notify ]
       │ (instant 201 queued)
       ▼
 [ Redis Stream ] ──(Worker Pool)──► [ dispatch.Executor ]
                                             │
      ┌──────────────────────────────────────┴──────────────────────────────────────┐
      │                                                                             │
      ▼ (Level 1: Retry)                                                            ▼ (Level 3: Circuit Breaker)
 [ Provider Send ] ◄── 3x Exponential Backoff                                  [ health.Registry ]
      │                                                                             │
      ├─ Failed after 3 retries?                                                    ├─ 5 consecutive failures?
      │  └─► (Level 2: Fallback)                                                    │  └─► Open circuit for 60s
      │      Try secondary channel in "auto" mode                                   │      Skip provider instantly in "auto"
      │      (Discord ──► Resend Email)                                             │      Probe with 1 trial request after 60s
      │                                                                             │
      └─ Worker process crashed / 3 worker attempts failed?                         └─────────────────────────────────────┘
         └─► (Level 4: Dead-Letter Queue)
             Move to relayhub:notifications:dlq
             Postgres status = "dead_letter"
             │
             └─► (Level 5: Replay Endpoint)
                 POST /v1/notify/:id/replay resets to queued and retries
```

1. **Level 1 — Per-Provider Retries (`retry.WithRetry`):** Every provider invocation retries up to 3 times with exponential back-off before reporting failure.
2. **Level 2 — Channel Fallback (`channel: "auto"`):** If the primary channel (Discord) fails after its retries, the engine automatically attempts the secondary channel (Resend Email).
3. **Level 3 — In-Memory Circuit Breaker (`health.Registry`):** Tracks consecutive failures per provider. After 5 consecutive failures, the provider circuit opens for 60 seconds. While open, `auto` routing bypasses the down provider immediately without waiting for timeouts. After 60 seconds, a single probe request is allowed through (half-open state); if it succeeds, the circuit closes.
4. **Level 4 — Dead-Letter Queue (DLQ):** Unacknowledged jobs (e.g. worker crashes) are reclaimed via Redis `XAUTOCLAIM`. After 3 total worker-level attempts, the job is promoted to the DLQ stream (`relayhub:notifications:dlq`) and marked `dead_letter` in Postgres to protect worker capacity.
5. **Level 5 — Manual Replay (`POST /v1/notify/:id/replay`):** Operators or tenants can inspect dead-lettered items via `GET /v1/notify/dead-letter` and re-trigger delivery once issues are resolved.

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

## React Dashboard (`/web`)

RelayHub includes an internal engineering testing dashboard built with **Vite + React + TypeScript**. It allows developers to visually test notification dispatching, track queue-to-delivery progression in real time, and audit tenant delivery history.

### Running the Dashboard

1. **Ensure the Go backend is running:**
   ```bash
   docker compose up -d
   # or locally: go run ./cmd/server
   ```

2. **Start the React dev server:**
   ```bash
   cd web
   npm install
   npm run dev
   ```

3. Open **http://localhost:5173** in your browser.

### Connecting to Backend

The frontend communicates with the backend via `VITE_API_BASE_URL` (default: `http://localhost:8080`). All requests are routed through `/web/src/api/client.ts` which automatically attaches the `X-API-Key` header from stored authentication state.

### Authentication & Storage Tradeoff Note

> ℹ️ **Security Note on API Key Storage:**
> The dashboard stores the tenant API key in browser `localStorage` + in-memory state. This design choice is **acceptable and intended** because the RelayHub Dashboard is a developer testing / internal engineering tool, not a public multi-tenant SaaS authentication client. In production deployments, standard session cookies or backend-proxied tokens would be used instead.

---

## Roadmap

- **Phase 1** ✅ Core engine — Discord provider, Email provider, delivery logs, retry, fallback, idempotency
- **Phase 2** ✅ Multi-tenancy — API key auth, per-tenant data isolation, rate limiting (100/day), usage stats
- **Phase 3** ✅ Templates, scheduled sends (`send_at`), outbound webhooks (HMAC-signed), SMTP provider
- **Phase 4** ✅ Redis Streams queue engine, worker pool, dead-letter queue, provider health circuit breakers
- **Phase 5** ✅ Full React Dashboard — Overview, Send, Logs, Templates, Webhooks, Dead Letter, Provider Health, Usage
