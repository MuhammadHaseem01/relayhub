# ⚡ RelayHub

> **A high-performance, self-hostable, multi-tenant notification delivery service.**

RelayHub is a unified notification platform built with **Go**, **Redis Streams**, and **PostgreSQL**. Instead of integrating multiple notification providers into your backend services, RelayHub exposes a single API endpoint that handles multi-channel dispatching (**Discord Webhooks**, **Resend Email**, and **SMTP**), automatic retries, intelligent fallback chains, rate limiting, dynamic templates, scheduled sends, dead-letter queue management, and HMAC-signed outbound webhooks.

Includes a modern, dark-mode **React + TypeScript Control Panel Dashboard** to test dispatches, monitor live queue status, manage templates, inspect raw API payloads, and track provider health.

---

## 🌟 Key Features

- ⚡ **Multi-Channel Provider System**: Unified API supporting Discord Webhooks, Resend Email API, and standard net/smtp transport.
- 🔄 **Intelligent Auto-Fallback & Circuit Breakers**: Automatically tracks provider health. If a primary channel (e.g. Discord) fails, RelayHub seamlessly falls back to Email/SMTP without dropping notifications.
- 🚀 **Async Redis Streams Architecture**: Non-blocking `/v1/notify` HTTP response ingestion backed by a consumer group worker pool for high-throughput background processing.
- 💀 **Dead-Letter Queue (DLQ) & Replay**: Notifications failing max worker attempts are safely promoted to the DLQ (`relayhub:notifications:dlq`) for manual audit and one-click replay.
- 🔑 **Multi-Tenancy & Rate Limiting**: Tenant API key authentication, per-tenant data isolation, and rolling 24-hour rate limit quotas.
- 📝 **Dynamic Handlebars Templates**: CRUD template engine with live client-side variable syntax validation.
- ⏰ **Scheduled Sends**: Delayed notification dispatching (`send_at`) managed by a multi-instance safe background scheduler.
- 🪝 **HMAC-Signed Outbound Webhooks**: Pushes real-time delivery event payloads (`notification.delivered` / `notification.failed`) to your backend with HMAC-SHA256 signature verification.
- 🖥️ **React + TypeScript Management Dashboard**: Dark-mode control panel to send test notifications, track live queue-to-delivery status, inspect raw API responses, audit delivery logs, and monitor circuit breaker health.

---

## 🏗️ Architecture Overview

```
                        ┌─────────────────────────────────────────┐
                        │      React Control Panel Dashboard      │
                        └────────────────────┬────────────────────┘
                                             │ HTTP / CORS
                                             ▼
┌──────────────────┐               ┌──────────────────┐
│  Client Apps     ├──────────────►│   RelayHub API   │
│  (HTTP / API)    │  POST /notify │   (Go Router)    │
└──────────────────┘               └─────────┬────────┘
                                             │
                       ┌─────────────────────┴─────────────────────┐
                       │                                           │
                       ▼ Async Job                                 ▼ Scheduled Job
            ┌─────────────────────┐                     ┌─────────────────────┐
            │    Redis Streams    │                     │     PostgreSQL      │
            │ (Consumer Workers)  │                     │   (Database Store)  │
            └──────────┬──────────┘                     └──────────┬──────────┘
                       │                                           │
                       └─────────────────────┬─────────────────────┘
                                             │
                                             ▼
                             ┌──────────────────────────────┐
                             │    Dispatch Engine           │
                             │ (Fallback + Circuit Breaker) │
                             └───────────────┬──────────────┘
                                             │
            ┌────────────────────────────────┼────────────────────────────────┐
            ▼                                ▼                                ▼
   ┌─────────────────┐              ┌─────────────────┐              ┌─────────────────┐
   │ Discord Webhook │              │  Resend Email   │              │   SMTP Server   │
   └─────────────────┘              └─────────────────┘              └─────────────────┘
                                             │
                                             ▼
                             ┌──────────────────────────────┐
                             │  HMAC Signed Outbound        │
                             │  Webhook Event Push          │
                             └──────────────────────────────┘
```

---

## 🚀 Quick Start

### Prerequisites

- [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)
- Alternatively: Go 1.22+, PostgreSQL 14+, and Redis 7+

### 1. Clone & Start Services

```bash
# Clone the repository
git clone https://github.com/MuhammadHaseem01/relayhub.git
cd relayhub

# Copy environment config
cp .env.example .env

# Launch PostgreSQL, Redis, and RelayHub API Server
docker compose up -d --build
```

The server will be available at `http://localhost:8080`.

Verify health check:
```bash
curl http://localhost:8080/v1/health
# Response: {"status":"ok","timestamp":"2026-07-30T16:00:00Z"}
```

---

### 2. Launch Management Dashboard

The interactive React dashboard runs in the `/web` directory.

```bash
cd web
npm install
npm run dev
```

Open **`http://localhost:5173`** in your browser.

1. Select **"Register Tenant"**, enter an application name, and copy your generated API key (`rh_...`).
2. Paste the key to authenticate and access the full control panel!

---

## 📖 API Reference & Examples

All requests except public health checks require the `X-API-Key` header.

### 1. Register a Tenant Account

```http
POST /v1/tenants
Content-Type: application/json

{
  "name": "Acme SaaS Production"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "tenant_id": "ca9a53bb-7d88-4c12-b912-3a7f8e0b1c2d",
    "api_key": "rh_a3f9c2d1..."
  }
}
```
> ⚠️ **Save `api_key` immediately.** It is returned only once upon creation.

---

### 2. Send a Notification (Auto Fallback)

Send a notification using automatic multi-channel fallback (tries Discord first; if it fails or is unhealthy, routes to Email/SMTP).

```http
POST /v1/notify
X-API-Key: rh_a3f9c2d1...
Content-Type: application/json

{
  "channel": "auto",
  "discord_recipient": "https://discord.com/api/webhooks/12345/abcde",
  "email_recipient": "user@example.com",
  "message": "Your backup has completed successfully!",
  "idempotency_key": "idem-backup-complete-20260730"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "request_id": "9f8e7d6c-5b4a-3f2e-1d0c-9b8a7f6e5d4c",
    "status": "queued",
    "channel": "auto"
  }
}
```

---

### 3. Send via Template & Variables

```http
POST /v1/notify
X-API-Key: rh_a3f9c2d1...
Content-Type: application/json

{
  "channel": "email",
  "recipient": "customer@example.com",
  "template": "order_confirmation",
  "variables": {
    "customer_name": "Jane Doe",
    "order_id": "ORD-98214"
  }
}
```

---

### 4. Schedule a Delayed Notification

```http
POST /v1/notify
X-API-Key: rh_a3f9c2d1...
Content-Type: application/json

{
  "channel": "discord",
  "recipient": "https://discord.com/api/webhooks/12345/abcde",
  "message": "Reminder: Scheduled maintenance starts in 10 minutes.",
  "send_at": "2026-07-30T18:00:00Z"
}
```

---

### 5. Check Notification Status & Delivery Log

```http
GET /v1/notify/9f8e7d6c-5b4a-3f2e-1d0c-9b8a7f6e5d4c
X-API-Key: rh_a3f9c2d1...
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 142,
    "tenant_id": "ca9a53bb-...",
    "request_id": "9f8e7d6c-...",
    "channel": "auto",
    "recipient": "user@example.com",
    "status": "delivered",
    "attempts": 2,
    "fallback_used": true,
    "created_at": "2026-07-30T16:05:00Z"
  }
}
```

---

### 6. Configure Outbound Webhooks

Receive real-time push events when notifications reach a terminal state (`notification.delivered` or `notification.failed`).

```http
PUT /v1/webhook
X-API-Key: rh_a3f9c2d1...
Content-Type: application/json

{
  "webhook_url": "https://api.yourdomain.com/webhooks/relayhub"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "webhook_url": "https://api.yourdomain.com/webhooks/relayhub",
    "webhook_secret": "3a7f8e9d0c1b2a3f4e5d6c7b8a9f0e1d"
  }
}
```

#### Signature Verification Example (Go)

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func VerifySignature(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

---

### 7. Replay Dead-Lettered Notifications

```http
POST /v1/notify/9f8e7d6c-5b4a-3f2e-1d0c-9b8a7f6e5d4c/replay
X-API-Key: rh_a3f9c2d1...
```

**Response:**
```json
{
  "success": true,
  "data": {
    "request_id": "9f8e7d6c-5b4a-3f2e-1d0c-9b8a7f6e5d4c",
    "status": "queued"
  }
}
```

---

## ⚙️ Environment Variables

Configuration is handled via environment variables (or `.env` file):

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server HTTP port |
| `DATABASE_URL` | `postgres://relayhub:secret@localhost:5432/relayhub?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection URL for Streams queue |
| `RESEND_API_KEY` | `""` | Resend API key for Email channel |
| `SMTP_HOST` | `""` | SMTP server host (e.g. `smtp.mailtrap.io`) |
| `SMTP_PORT` | `587` | SMTP server port |
| `SMTP_USERNAME` | `""` | SMTP authentication username |
| `SMTP_PASSWORD` | `""` | SMTP authentication password |
| `SMTP_FROM` | `notifications@relayhub.internal` | Default From email address for SMTP |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | Allowed CORS origin for Dashboard dev server |

---

## 📂 Project Structure

```
relayhub/
├── cmd/
│   └── server/
│       └── main.go                 # Application entrypoint & dependency wiring
├── internal/
│   ├── dispatch/                   # Core notification dispatch executor
│   ├── health/                     # Provider health circuit breakers
│   ├── middleware/                 # API Key authentication & rate limiting
│   ├── providers/                  # Provider adapters (Discord, Resend, SMTP)
│   ├── queue/                      # Redis Streams queue & worker pool
│   ├── router/                     # HTTP router, middleware, and handlers
│   ├── scheduler/                  # Background cron scheduler for send_at
│   ├── store/                      # PostgreSQL data layer & migrations
│   ├── template/                   # Handlebars variable substitution engine
│   ├── webhook/                    # Outbound HMAC-signed webhook delivery
│   └── worker/                     # Worker pool execution loop
├── web/                            # React + TypeScript Vite Control Panel
│   ├── src/
│   │   ├── api/                    # Type-safe API client wrapper
│   │   ├── components/             # Reusable UI components & modals
│   │   ├── pages/                  # Dashboard, Send, Logs, Templates, Webhooks, DLQ, Health, Usage
│   │   └── styles/                 # Theme tokens & global dark mode CSS
│   ├── package.json
│   └── vite.config.ts
├── docker-compose.yml              # Local infrastructure orchestration
├── Dockerfile                      # Multi-stage Docker build
└── README.md
```

---

## 📜 License

This project is licensed under the **MIT License**.
