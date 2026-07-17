# Architecture

This template now runs the Cargonex backend contract in the template-style Go layout.

## Active Shape

- `main.go`: env loading, logrus setup, and server startup.
- `src/config`: environment-backed runtime configuration with Cargonex defaults.
- `src/database`: Postgres connection setup from `DATABASE_URL`.
- `src/database/store`: persistence operations, owned-resource CRUD, status histories, ledgers, resource resets, and tour side effects.
- `src/controllers/*_service`: service interfaces.
- `src/controllers/*_service/*_service_impl`: concrete service implementations.
- `src/router`: explicit Gin route registration for all Cargonex `/api` routes, auth checks, validation, response formatting, and HTTP handlers.
- `src/server`: HTTP server lifecycle and graceful shutdown.
- `cmd/migrate`: SQL migration runner for `database/migrations`.
- `cmd/seed`: seed super-admin user.
- `cmd/createorguser`: create or update an organization admin.
- `cmd/contracttest`: compare Cargonex-compatible responses across two running backends.

## Request Flow

1. `main.go` loads env and calls `server.Start(config.Load())`.
2. `src/server` opens Postgres and builds `router.NewRouter`.
3. `src/router/module.go` registers every supported Cargonex route with Gin.
4. Router handlers validate request bodies and auth tokens.
5. Handlers call service interfaces in `src/controllers`.
6. Service implementations coordinate database work through `src/database/store`.
7. Responses keep the frontend-compatible `{ success, message, ... }` envelope.

## Compatibility Rules

- Default port is `5000`.
- All routes are under `/api`.
- `POST /api/user/login` is public.
- All other API routes require the Cargonex HMAC token.
- Auth tokens are accepted from `Authorization`, `Token`, or `X-Access-Token`.
- Trailing slashes are normalized.
- CORS defaults to `http://localhost:3000`.

Inactive delivery-template files are kept locally under Go-ignored and gitignored `_legacy_delivery` folders for reference only. They are not part of the active Go package tree.
