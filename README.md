# Railzway

Railzway is an open-source billing engine for teams that want to model subscriptions, usage-based pricing, rating, proration, and invoicing. It focuses on determining what should be billed; payment execution is handled by integrations.

## Status

Railzway is under active development. Some flows and APIs are still evolving, so treat this as a working preview rather than a finished product.

## Quick Start (Local, from source)

### Prerequisites

Tested with:

- Go 1.25
- Node.js 20
- PostgreSQL 16
- Redis 7

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

```bash
migrate -path db/migrations -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" up
```

### 4) Build admin UI

If you want the admin UI served by the Go backend, build the admin UI first.
Skip this step if you only need the API, or prefer the dev server (see step 5).

```bash
pnpm --dir apps/admin install
pnpm --dir apps/admin build
```

### 5) Run admin backend

```bash
go run cmd/admin/main.go
```

Admin backend default: `http://localhost:8080`

If you prefer a frontend dev server instead:

```bash
pnpm --dir apps/admin dev
```

Admin UI (dev) default: `http://localhost:5173`

## Project Structure

```
apps/            # Frontend apps (admin, invoice, checkout, customer)
cmd/             # Go binaries (admin, scheduler, api, checkout, customer)
config/          # Local env examples and runtime configs
db/migrations/   # Database migrations
deployment/      # Docker compose, build scripts
docs/            # Long-form documentation and diagrams
internal/        # Core domains, services, repositories, transport
packages/        # Shared UI/design-system packages
```

Where to start:
- **Product logic**: `internal/*/service`
- **API handlers**: `internal/admin/transport/http`
- **Scheduler jobs**: `internal/*/scheduler`
- **Admin UI**: `apps/admin/src`

## Apps & Binaries

**Frontend apps (apps/):**
- `apps/admin` – Admin console (React/Vite)
- `apps/invoice` – Invoice UI
- `apps/checkout` – Checkout UI
- `apps/customer` – Customer portal

**Go binaries (cmd/):**
- `cmd/admin` – Admin backend (serves admin API + static UI)
- `cmd/scheduler` – Background jobs
- `cmd/api` – Public API (API-key auth)
- `cmd/checkout` – Checkout service host
- `cmd/customer` – Customer portal host

## Documentation

Long-form documentation lives in `docs/`. Start from `docs/` if you want deeper design context.

## Notes

- Org-scoped resources require the `X-Org-ID` header.
- The admin UI sets this after you choose an organization.
- The public API (API-key auth) is evolving; documentation will be published when stable.
- If you do not have the `migrate` CLI:
  - `brew install golang-migrate`, or
  - `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

## Contributing

If you plan to contribute, open an issue with a short proposal or a reproduction first so changes align with the current roadmap. See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.
