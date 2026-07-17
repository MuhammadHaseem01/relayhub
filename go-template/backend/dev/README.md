# Cargonex Development

This folder contains local Docker configuration for the Cargonex Go backend template.

## Compose

Start Postgres and the Go API:

```bash
docker compose -f dev/docker-compose.yml up --build -d
```

Stop the stack:

```bash
docker compose -f dev/docker-compose.yml down
```

Run migrations through Docker after the stack is up:

```bash
docker compose -f dev/docker-compose.migrate.yml run --rm migrate
```

## Local Run

From the backend root:

```bash
make migrate
make seed
make run
```

The API defaults to `http://localhost:5000/api`.
