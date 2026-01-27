# Deployment Guide

This guide explains how to deploy Railzway services using Docker and Docker Compose.

## 🐳 Docker Images

We publish the following images to GitHub Container Registry (GHCR):

| Service | Image | Description |
| :--- | :--- | :--- |
| **Admin (Monolith)** | `ghcr.io/smallbiznis/railzway/railzway-admin` | Core Monolith. Serving Admin UI + API. |
| **Invoice** | `ghcr.io/smallbiznis/railzway/railzway-invoice` | Customer-facing Invoice Checkout UI. |
| **Scheduler** | `ghcr.io/smallbiznis/railzway/railzway-scheduler` | Background workers (Rating, Invoicing). No UI. |
| **Migration** | `ghcr.io/smallbiznis/railzway/railzway-migration` | One-off container for running database migrations. |

## 🚀 Running with Docker Compose

Create a `docker-compose.yml` file with the following content:

```yaml
services:
  # Database
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: railzway
      POSTGRES_PASSWORD: password
      POSTGRES_DB: railzway
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  # Infrastructure
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  # -----------------------------------------------------
  # Core Services
  # -----------------------------------------------------
  
  # 1. Admin Dashboard (UI + API)
  admin:
    image: ghcr.io/smallbiznis/railzway/railzway-admin:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis

  # 2. Scheduler (Background Jobs)
  scheduler:
    image: ghcr.io/smallbiznis/railzway/railzway-scheduler:latest
    environment:
      - ENABLED_JOBS=billing,usage,invoice # Run all jobs
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis

  # 3. Public Invoice Service
  invoice:
    image: ghcr.io/smallbiznis/railzway/railzway-invoice:latest
    ports:
      - "3000:8080" # Exposed on port 3000
    environment:
      - PORT=8080
      - API_URL=http://admin:8080 # Points to Admin API for data
    depends_on:
      - admin

volumes:
  postgres_data:
```

### Start the Stack
```bash
docker-compose up -d
```

## 📦 Running Individual Containers

If you prefer `docker run`:

### 1. Run Admin Service
```bash
docker run -d \
  --name railzway-admin \
  -p 8080:8080 \
  -e DB_HOST=host.docker.internal \
  -e REDIS_HOST=host.docker.internal \
  ghcr.io/smallbiznis/railzway/railzway-admin:latest
```

### 2. Run Scheduler
```bash
docker run -d \
  --name railzway-scheduler \
  -e DB_HOST=host.docker.internal \
  -e REDIS_HOST=host.docker.internal \
  ghcr.io/smallbiznis/railzway/railzway-scheduler:latest
```

> Note: `host.docker.internal` allows the container to access services running on your host machine (like local Postgres).

## Environment Variables

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | HTTP Port to listen on | `8080` |
| `DB_HOST` | Postgres Host | `localhost` |
| `DB_USER` | Postgres User | `postgres` |
| `DB_NAME` | Postgres DB Name | `postgres` |
| `REDIS_HOST` | Redis Host | `localhost` |
| `ENABLED_JOBS` | Comma-separated list of jobs (Scheduler only) | All jobs |
| `ENSURE_DEFAULT_ORG_AND_USER` | Auto-create default org + admin on startup (OSS/dev) | `true` |
| `BOOTSTRAP_DEFAULT_ORG_ID` | Explicit org ID for auto-bootstrap | `0` |
| `BOOTSTRAP_DEFAULT_ORG_NAME` | Explicit org name for auto-bootstrap | (empty) |
| `BOOTSTRAP_DEFAULT_ORG_SLUG` | Explicit org slug for auto-bootstrap | (empty) |
| `BOOTSTRAP_ADMIN_EMAIL` | Admin email for auto-bootstrap | `admin@railzway.com` |
| `BOOTSTRAP_ADMIN_PASSWORD` | Admin password for auto-bootstrap | `admin` |

> **Note:** Use `BOOTSTRAP_DEFAULT_ORG_*` for both OSS and Cloud.

## Org Activation Boundary (Cloud vs Core)

These rules are **hard constraints** for a Cloud-ready control-plane without leaking domain logic into the core engine:

1. **OrgGate never joins `organizations`.**  
   All enforcement reads **only** `org_bootstrap_state` by `org_id`.  
   If you need to log `org_name` or `slug`, do it in the handler layer by enrichment, **not** inside the gate.
2. **No implicit activation path.**  
   `CreateOrganization(...)` must create `org_bootstrap_state = initializing` (or `active` only via backfill/compat).  
   Activation must be explicit via `ActivateOrganization(org_id)` (or control-plane event) and irreversible.
3. **Slug is UX-only, never a billing key.**  
   All billing/scheduler/ledger logic uses `org_id`.  
   Slug lookup is allowed only at the edge (convert slug → org_id), then the core uses `org_id` exclusively.

## Testing: Migration Guard & Org Activation

> **Run these steps on a dev database only.**

### 1) Migration Guard (Schema Gate)

1. Run migrations:
   ```bash
   go run ./cmd/railzway migrate
   ```
2. Verify `system_bootstrap_state` is `active`:
   ```sql
   SELECT status, schema_version, activated_at FROM system_bootstrap_state;
   ```
3. Start services (should succeed):
   ```bash
   go run ./cmd/railzway serve
   go run ./cmd/railzway scheduler
   ```
4. Force a failure state (dev only):
   ```sql
   UPDATE system_bootstrap_state SET status = 'initializing';
   ```
5. Start services again (should **fail fast** at schema gate):
   ```bash
   go run ./cmd/railzway serve
   go run ./cmd/railzway scheduler
   ```

### 2) Org Activation Gate

1. Create an org and initialize bootstrap state (dev only):
   ```sql
   INSERT INTO organizations (id, name, slug) VALUES (123, 'Test Org', 'test-org');
   INSERT INTO org_bootstrap_state (org_id, status, created_at)
   VALUES (123, 'initializing', NOW());
   ```
2. Try a billing or scheduler action for `org_id = 123`  
   Expected: **denied** (org is not active).
3. Activate explicitly:
   ```sql
   UPDATE org_bootstrap_state
   SET status = 'active', activated_at = NOW()
   WHERE org_id = 123;
   ```
4. Retry the same billing/scheduler action  
   Expected: **allowed**.
