# Cutover Plan

Use this checklist when the new Go backend is ready to replace NestJS.

## Preconditions

- `make fmt` passes without changing unexpected files.
- `make tidy` produces stable `go.mod` and `go.sum`.
- `make test` passes.
- `make migrate` creates a clean database from `database/migrations`.
- `make seed` creates a login-capable admin user.
- Contract tests pass against NestJS and the new Go backend using equivalent database snapshots.

## Parallel Run

1. Run NestJS on port `5000`.
2. Run this template backend on port `5001` with `make run-side-by-side`.
3. Run the contract test suite with the same login credentials or shared token.
4. Compare high-risk workflows manually from the frontend:
   - Login and user management.
   - Driver and vehicle status updates.
   - Tour create/update/complete/delete.
   - Vehicle maintenance create/update/delete.
   - Ledger views and petrol pump payment updates.

## Switch

1. Stop NestJS on port `5000`.
2. Run the Go backend on its default port `5000` with `make run`.
3. Keep the frontend API base URL unchanged.
4. Keep the NestJS app available for rollback during the first verification window.
5. Watch logs for auth failures, validation mismatches, and ledger/status side-effect errors.
6. After verification, stop deploying NestJS for this backend.

## Legacy Code

- `../go-backend` and `../go-cargonex-backend` are migration references only.
- This `go-template/backend` directory is the active Go backend target.
