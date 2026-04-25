# Railzway

Railzway is an open-source **Billing Computation Engine** developed as a solo project with AI assistance. It focuses on solving the core logic of SaaS billing through clear system boundaries, pragmatic architecture, and financial correctness.

> [!NOTE]
> **Why Railzway?** This project is an investigation into building a billing system that prioritizes **Financial Integrity** over convenience. By separating billing computation from payment collection, I've implemented a **Double-Entry Ledger** logic and **Reconciliation** prototype to explore how to build a 100% auditable billing core as a solo developer.

It ingests usage events, rates them against pricing models, and generates ledger-linked invoices. It's a work-in-progress effort to learn and implement mission-critical billing logic at scale.

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

## Architecture Stance

Railzway is a pragmatic monorepo. It is not a strict Clean Architecture or DDD-only codebase.

The repository favors clear boundaries, navigable code, delivery speed, and operational clarity over architectural purity. Some parts are organized by domain, while others are organized by product surface and runtime topology. That tradeoff is intentional.

If you want the longer explanation, see [`docs/architecture/repository-structure-philosophy.md`](./docs/architecture/repository-structure-philosophy.md).

## Quick Start (Local, from source)

Unless noted otherwise, commands below are run from the repository root.

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

Binary-specific defaults are also available under `config/`, for example:

- `config/base.defaults.yml`
- `config/admin/defaults.yml`

Runtime precedence is:

1. built-in defaults
2. `config/base.defaults.yml`
3. `config/<binary>/defaults.yml`
4. environment variables

### 2) Start dependencies

```bash
docker compose -f deployment/docker/docker-compose.yml up -d postgres redis
```

### 3) Run migrations

```bash
go run ./cmd/migrate up --database-url "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
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
go run ./cmd/admin
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
- `cmd/admin` – Admin host binary used for both backend-only and all-in-one admin packaging
- `cmd/scheduler` – Background jobs
- `cmd/api` – Public API (API-key auth)
- `cmd/checkout` – Checkout service host
- `cmd/customer` – Customer portal host
- `cmd/migrate` – Database migration runner (`railzway-migrate`)

Browser-facing surfaces can also be deployed with the UI hosted separately from compute. See the deployment guidance in the architecture docs for recommended same-origin and shared-API topologies.

Database migrations are executed through the repo-native `railzway-migrate` binary in `cmd/migrate`. This keeps migration commands consistent across local development, CI, and containerized execution.

Container image naming is explicit by topology:
- `railzway-admin-all-in-one` = admin BEFE bundle (backend + bundled admin UI)
- `railzway-admin-api` = backend-only admin host

On pushes to `main`, CI publishes the Docker targets that exist today to GitHub Container Registry (`ghcr.io`): `railzway-admin-all-in-one`, `railzway-admin-api`, `railzway-scheduler`, `railzway-api`, and `railzway-migrate`.

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
- [`docs/architecture/coupons-and-promotions.md`](./docs/architecture/coupons-and-promotions.md) – Coupon, promotion code, segment, and discount-application domain model.
- [`docs/architecture/browser-surfaces-and-deployment.md`](./docs/architecture/browser-surfaces-and-deployment.md) – Browser surface topology, same-origin proxy recommendation, and shared API domain alternative.
- [`docs/architecture/repository-structure-philosophy.md`](./docs/architecture/repository-structure-philosophy.md) – Why the repo favors pragmatic modularity over strict architectural doctrine.

## Notes

- Org-scoped resources require the `X-Org-ID` header.
- The admin UI sets this after you choose an organization.
- The public API (API-key auth) is evolving; documentation will be published when stable.
- Coupon and promotion architecture: see [`docs/architecture/coupons-and-promotions.md`](./docs/architecture/coupons-and-promotions.md).
- Public API rate limits: see [`docs/api/rate-limits.md`](./docs/api/rate-limits.md).
- Migration helper examples:
  - `go run ./cmd/migrate up`
  - `go run ./cmd/migrate down 1`
  - `go run ./cmd/migrate steps -1`
  - `go run ./cmd/migrate force 20260317074922`
  - `go run ./cmd/migrate version`

## Contributing

If you plan to contribute, open an issue with a short proposal or a reproduction first so changes align with the current roadmap. See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.
