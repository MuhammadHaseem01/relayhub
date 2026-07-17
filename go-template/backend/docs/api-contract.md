# API Contract Checklist

The NestJS backend remains the behavior source of truth until contract tests pass.

## Global Behavior

- Default local port is `5000`, matching the current NestJS backend used by the frontend.
- Prefix all backend routes with `/api`.
- Allow CORS from `http://localhost:3000` by default.
- Return successful object responses with `success: true` and a `message` field when one is not already present.
- Return errors as `{ "message": ..., "success": false }`.
- Keep `POST /api/user/login` public.
- Require the custom HMAC token for all other API routes.
- Accept auth tokens from `Authorization`, `Token`, or `X-Access-Token`.

## Routes

User:

- `POST /api/user/login`
- `POST /api/user/register`
- `POST /api/user/add`
- `GET /api/user`
- `GET /api/user/:id`
- `PATCH /api/user/:id`
- `DELETE /api/user/:id`

Client:

- `POST /api/client/add`
- `GET /api/client`
- `GET /api/client/:id`
- `PATCH /api/client/:id`
- `DELETE /api/client/:id`

Arrange vehicle:

- `POST /api/arrange-vehicle/add`
- `GET /api/arrange-vehicle`
- `GET /api/arrange-vehicle/:id`
- `PATCH /api/arrange-vehicle/:id`
- `DELETE /api/arrange-vehicle/:id`

Inventory:

- `POST /api/inventory/add`
- `GET /api/inventory`
- `GET /api/inventory/:id`
- `PATCH /api/inventory/:id`
- `DELETE /api/inventory/:id`

Vehicle:

- `POST /api/vehicle/add`
- `GET /api/vehicle`
- `GET /api/vehicle/:id`
- `PATCH /api/vehicle/:id`
- `DELETE /api/vehicle/:id`

Driver:

- `POST /api/driver/add`
- `GET /api/driver`
- `PATCH /api/driver/:id`
- `DELETE /api/driver/:id`

Vehicle maintenance:

- `POST /api/vehicle-maintenance/add`
- `GET /api/vehicle-maintenance`
- `GET /api/vehicle-maintenance/:id`
- `PATCH /api/vehicle-maintenance/:id`
- `PATCH /api/vehicle-maintenance/update`
- `PATCH /api/vehicle-maintenance/update/:id`
- `DELETE /api/vehicle-maintenance/:id`

Tour damage:

- `POST /api/tour-damage/add`
- `GET /api/tour-damage`
- `GET /api/tour-damage/:id`
- `PATCH /api/tour-damage/:id`
- `DELETE /api/tour-damage/:id`

Tour deduction:

- `POST /api/tour-deduction/add`
- `GET /api/tour-deduction`
- `PATCH /api/tour-deduction/:id`
- `DELETE /api/tour-deduction/:id`

Tour:

- `POST /api/tour/add`
- `GET /api/tour`
- `GET /api/tour/:id`
- `PATCH /api/tour/:id`
- `DELETE /api/tour/:id`

Ledger:

- `GET /api/ledger`
- `GET /api/ledger/:id`

## Parity-Sensitive Side Effects

- User registration must enforce email uniqueness, bcrypt hashing, ownership, organization ID generation, roles, permissions, and active/inactive status.
- Vehicle and driver creation/update must create status history rows.
- Vehicle and driver list/get responses must include status history.
- Vehicle maintenance create/update/delete must sync vehicle-maintenance ledger rows.
- Tour create/update/delete must sync tour fuel ledgers.
- Tour completion and deletion of active/planned tours must reset assigned driver and vehicle status.
- Tour deduction create/delete must update related tour status and resources.
- Non-numeric `PATCH /api/tour/:id` must update fuel payment status by petrol pump slug.
