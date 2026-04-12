# Railzway

Railzway is an open-source billing engine that aims to help teams model subscriptions, usage-based pricing, ratings, proration, and invoicing. The project is under active development and the feature surface is evolving.

## Status

Railzway is in active development. Some flows are still being refined, and you may encounter incomplete screens or evolving APIs. If you want to try it, treat the current build as a working preview rather than a finished product.

## What’s Inside

- **Admin backend** (Go): admin API + UI host.
- **Scheduler** (Go): background jobs (rating, reconciliation, close-period, etc.).
- **Public API** (Go, optional): endpoints intended for API-key based integrations.
- **Admin UI** (Vite/React): web console for billing operations.

## Quick Start (Docker Compose)

This path uses prebuilt images when they are available.

1. Copy environment files:

```bash
cp config/docker/admin.env.example config/docker/admin.env
cp config/docker/scheduler.env.example config/docker/scheduler.env
```

2. (Optional) Update secrets in `config/docker/*.env` for local testing.

3. Start services:

```bash
cd deployment/docker
docker compose up -d
```

4. Open the admin UI:

- `http://localhost:8080`

## Local Development (from source)

### Prerequisites

- Go 1.25+
- Node.js 20+
- PostgreSQL 16+
- Redis 7+

### 1) Environment

```bash
cp .env.example .env
```

At minimum, set:

- `SESSION_SECRET`
- `APPS_CREDENTIALS_KEY` (base64 32 bytes)
- `RAILZWAY_ORG_NAME`, `RAILZWAY_USER_EMAIL`, `RAILZWAY_USER_PASSWORD`

### 2) Start dependencies

```bash
cd deployment/docker
docker compose up -d postgres redis
```

### 3) Run migrations (manual)

Install the `migrate` CLI (golang-migrate) if you do not have it yet:

```bash
brew install golang-migrate
```

or:

```bash
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Then run migrations:

```bash
migrate -path db/migrations -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" up
```

### 4) Run admin backend

```bash
go run cmd/admin/main.go
```

### 5) Run admin UI (dev)

```bash
pnpm --dir apps/admin install
pnpm --dir apps/admin dev
```

Admin UI default: `http://localhost:5173`

## Notes

- Admin APIs are served under `/admin/v1`.
- Org-scoped resources require the `X-Org-ID` header. The admin UI manages this after you pick an organization.
- The public API (API-key auth) is intended for integrations and may be enabled separately.

## Contributing

This project is evolving quickly. If you plan to contribute, open an issue with a short proposal or a reproduction first so changes align with the current roadmap.
