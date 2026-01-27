# Deployment Guide

This guide explains how to deploy Railzway services using the **Unified Docker Strategy**.

## 🐳 Docker Images & Build Strategy

Railzway uses a **single multi-stage `Dockerfile`** at the project root. This single file can build specialized images for all system components.

### 1. Unified `Dockerfile` Targets

To build a specific component manualy, use the `--target` flag:

```bash
# Admin Monolith (UI + API)
docker build --target railzway-admin -t railzway-admin .

# Invoice UI (Customer Facing)
docker build --target railzway-invoice -t railzway-invoice .

# Scheduler (Background Jobs)
docker build --target railzway-scheduler -t railzway-scheduler .

# Migration Runner (Tools)
docker build --target railzway-migration -t railzway-migration .
```

### 2. Published Images (GHCR)

We automatically build and publish these targets to GitHub Container Registry:

| Service | Image | Description |
| :--- | :--- | :--- |
| **Admin** | `ghcr.io/smallbiznis/railzway/railzway-admin` | Core Monolith. Serving Admin UI + API. |
| **Invoice** | `ghcr.io/smallbiznis/railzway/railzway-invoice` | Customer-facing Invoice Checkout UI. |
| **Scheduler** | `ghcr.io/smallbiznis/railzway/railzway-scheduler` | Background workers (Rating, Invoicing). No UI. |
| **Migration** | `ghcr.io/smallbiznis/railzway/railzway-migration` | One-off container for running database migrations. |

---

## 🚀 Running with Docker Compose

We support two primary workflows: **Local Development** (Build from Source) and **Production** (Pull from Registry).

### A. Local Development

The default `docker-compose.yml` in the root directory is configured for local development. It builds images directly from your local source code.

**1. Start Infrastructure & Build:**
```bash
docker-compose up -d --build postgres redis
```

**2. Run Migrations (First Time Only):**
```bash
# This compiles the migration target and runs schema updates
docker-compose run --rm migration
```

**3. Start Services:**
```bash
docker-compose up -d
```

**Access Points:**
- **Admin Dashboard**: [http://localhost:8080](http://localhost:8080)
- **Public Invoice UI**: [http://localhost:3000](http://localhost:3000)

### B. Production Deployment

For production, create a specific `docker-compose.prod.yml` that uses the pre-built images from GHCR.

**Example `docker-compose.prod.yml`**:

```yaml
version: "3.8"
services:
  admin:
    image: ghcr.io/smallbiznis/railzway/railzway-admin:latest
    restart: always
    environment:
      - PORT=8080
      - DB_HOST=postgres
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - REDIS_HOST=redis
    ports: ["8080:8080"]
    depends_on: [postgres, redis]

  invoice:
    image: ghcr.io/smallbiznis/railzway/railzway-invoice:latest
    restart: always
    environment:
      - API_URL=http://admin:8080
    ports: ["3000:8080"]
    depends_on: [admin]

  scheduler:
    image: ghcr.io/smallbiznis/railzway/railzway-scheduler:latest
    restart: always
    environment:
      - ENABLED_JOBS=billing,usage,invoice
    depends_on: [postgres, redis]

  # External Infra (Managed DB recommended for Prod)
  postgres:
    image: postgres:15-alpine
    volumes: [postgres_data:/var/lib/postgresql/data]
  
  redis:
    image: redis:7-alpine

volumes:
  postgres_data:
```

---

## 📦 Running Individual Containers (Docker Run)

If you need to verify or run a single container manually:

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

> **Note**: `host.docker.internal` is used to access services running on the host machine. On Linux, proceed with `--network host` or valid IP address configuration.

## Environment Variables

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | HTTP Port to listen on (Admin/Invoice). | `8080` |
| `DB_HOST` | Postgres Hostname. | `localhost` |
| `DB_USER` | Postgres Username. | `postgres` |
| `DB_PASSWORD` | Postgres Password. | (empty) |
| `DB_NAME` | Postgres Database Name. | `postgres` |
| `REDIS_HOST` | Redis Hostname. | `localhost` |
| `API_URL` | (Invoice Service Only) URL to the Admin API. | `http://admin:8080` |
| `ENABLED_JOBS` | (Scheduler Only) Comma-separated list of jobs. | All jobs |
