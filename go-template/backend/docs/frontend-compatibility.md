# Frontend Compatibility Contract

The Go backend is intended to be a drop-in replacement for the current NestJS backend used by the existing frontend.

## Base URL

Default local URL:

```text
http://localhost:5000
```

All API routes keep the same global prefix:

```text
/api
```

No frontend route rewrites should be required when switching from NestJS to Go.

Trailing slashes are normalized, so these are treated the same:

```text
/api/client
/api/client/
```

## Auth Middleware Compatibility

`POST /api/user/login` is public.

All other API routes require the same token formats accepted by the NestJS middleware:

```http
Authorization: Bearer <token>
```

or:

```http
Authorization: <token>
```

or:

```http
Token: <token>
```

or:

```http
X-Access-Token: <token>
```

The login response keeps the frontend-compatible token key:

```json
{
  "message": "Login Successfully",
  "success": true,
  "Token": "..."
}
```

## Response Format

Successful object responses include:

```json
{
  "message": "Request successful",
  "success": true
}
```

Successful non-object responses are wrapped as:

```json
{
  "message": "Request successful",
  "success": true,
  "data": "..."
}
```

Errors use:

```json
{
  "message": "Unauthorized",
  "success": false
}
```

Validation errors keep Nest-compatible string-array messages:

```json
{
  "message": [
    "email must be an email"
  ],
  "success": false
}
```

## CORS

Default allowed origin:

```text
http://localhost:3000
```

Override with:

```bash
CORS_ORIGIN=http://your-frontend-origin
```

CORS preflight accepts these frontend-facing headers:

```text
Authorization, Token, X-Access-Token, Content-Type, Accept, Origin
```

## Side-by-Side Testing

When comparing NestJS and Go together:

- Run NestJS on `5000`.
- Run Go on `5001` with `make run-side-by-side`.
- Run contract tests with `NEST_BASE_URL=http://localhost:5000` and `GO_BASE_URL=http://localhost:5001`.

When switching the frontend to Go:

- Stop NestJS.
- Run Go on default port `5000` with `make run`.
