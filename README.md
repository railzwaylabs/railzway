# Railzway

Railzway is an open-source billing computation engine for SaaS products.

It ingests usage events, rates them against pricing, applies subscription and proration rules, generates invoices, and records ledger transactions. Railzway computes what should be billed and stores the results so they can be reproduced and audited later.

It is for teams building SaaS billing systems with usage-based, tiered, or hybrid pricing. It is not a payment processor: providers such as Stripe or Xendit belong on the integration side, outside the billing core.

## What Railzway Is

- A billing engine for metered and subscription SaaS products
- A system for turning usage events and plan prices into rating results, invoice lines, and invoice totals
- A ledger-backed billing record so billing outputs can be inspected, reproduced, and audited
- A billing core that can integrate with payment providers instead of coupling billing logic to one processor

## What Railzway Is Not

- Not a payment gateway or payment processor
- Not the system that charges cards, settles funds, or moves money
- Not a checkout-only product with billing rules embedded inside a payment provider

## When to Use Railzway

- Usage-based SaaS billing where raw events must be metered, aggregated, and priced
- Subscription billing with mid-cycle changes, start/end windows, or proration
- Tiered, flat, or hybrid pricing models that are hard to express cleanly in a payment processor
- Invoice generation where each bill needs a clear audit trail from usage and pricing inputs to ledger entries

## Mental Model

`usage events -> rating -> rating results + usage aggregates -> draft invoice -> open invoice -> ledger -> payment adapter/provider`

Railzway owns the billing computation path from usage to invoice and ledger. Payment providers sit after that boundary as adapters that collect payment for an amount Railzway has already computed.

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

- [`docs/architecture/catalog-model.md`](./docs/architecture/catalog-model.md) – Product/Plan/Price mental model.
- [`docs/architecture/metering.md`](./docs/architecture/metering.md) – Usage events ingestion and aggregation logic.
- [`docs/architecture/ledger.md`](./docs/architecture/ledger.md) – Double-entry accounting and financial integrity.
- [`docs/architecture/invoicing.md`](./docs/architecture/invoicing.md) – Invoicing lifecycle and the Rating engine.
- [`docs/architecture/subscriptions.md`](./docs/architecture/subscriptions.md) – Subscription lifecycle and proration logic.
- [`docs/architecture/taxes-discounts.md`](./docs/architecture/taxes-discounts.md) – Global tax compliance and price adjustments.
- [`docs/architecture/reconciliation.md`](./docs/architecture/reconciliation.md) – Automated data integrity and cross-module verification.
- [`docs/architecture/security.md`](./docs/architecture/security.md) – Enterprise RBAC model and security governance.

## Notes

- Org-scoped resources require the `X-Org-ID` header.
- The admin UI sets this after you choose an organization.
- The public API (API-key auth) is evolving; documentation will be published when stable.
- Public API rate limits: see [`docs/api/rate-limits.md`](./docs/api/rate-limits.md).
- If you do not have the `migrate` CLI:
  - `brew install golang-migrate`, or
  - `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

## Contributing

If you plan to contribute, open an issue with a short proposal or a reproduction first so changes align with the current roadmap. See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.
