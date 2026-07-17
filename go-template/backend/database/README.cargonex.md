# Database

The current schema source of truth is the root Prisma schema and migration:

- `../../prisma/schema.prisma`
- `../../prisma/migrations/20260619175854_init/migration.sql`

This backend now also includes a Go-owned SQL migration at `database/migrations/0001_init.sql`.

For parity work, keep the Go backend pointed at a database created from the same schema. Do not rename quoted Postgres identifiers such as `"User"`, `"Tour"`, `"createdById"`, or Prisma enum names unless the NestJS backend is changed at the same time.

Planned follow-up:

- Move runtime schema checks, especially ledger-table creation, into migrations.
- Compare `0001_init.sql` against the root Prisma migration whenever the Nest schema changes.
