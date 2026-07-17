# Cargonex Go Backend Template

This template has been updated to run the Cargonex backend contract while keeping the Go template layout:

- `main.go` bootstrap with env loading and structured logging.
- `src/config` runtime configuration.
- `src/database` database setup.
- `src/router` Gin route registration.
- `src/controllers/*_service` service interfaces.
- `src/controllers/*_service/*_service_impl` service implementations.
- `src/database/store` Cargonex persistence and side-effect operations.
- `cmd/migrate`, `cmd/seed`, `cmd/createorguser`, and `cmd/contracttest`.

The API keeps the frontend-compatible Cargonex/NestJS contract:

- Default port: `5000`
- Base path: `/api`
- Default CORS origin: `http://localhost:3000`
- Login route: `POST /api/user/login`
- Token headers: `Authorization`, `Token`, or `X-Access-Token`
- Successful response envelope includes `success: true`
- Error response envelope includes `success: false`

## Local Commands

```bash
make fmt
make tidy
make test
make migrate
make seed
make create-org-user
make run
```

Run side by side with another backend on port `5000`:

```bash
make run-side-by-side
```

## Contract Test

```bash
NEST_BASE_URL=http://localhost:5000 \
GO_BASE_URL=http://localhost:5001 \
CONTRACT_LOGIN_EMAIL=admin@example.com \
CONTRACT_LOGIN_PASSWORD='password123' \
make contract
```

See `docs/` for the API contract and migration notes.
