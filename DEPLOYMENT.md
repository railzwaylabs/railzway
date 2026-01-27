# Deployment Guide

This guide explains how to **run** Railzway services using Docker Compose and Docker.

## 🚀 Quick Start (Docker Compose)

The easiest way to run Railzway locally is using the included `docker-compose.yml`.

### 1. Start Support Infrastructure
Start the database and cache services first:

```bash
docker-compose up -d postgres redis
```

### 2. Run Database Migrations
Initialize the database schema (run this once, or whenever schema changes):

```bash
docker-compose run --rm migration
```

### 3. Start Application Services
Start the Admin, Invoice, and Scheduler services:

```bash
docker-compose up -d
```

### 4. Access the Application
*   **Admin Dashboard**: [http://localhost:8080](http://localhost:8080)
*   **Public Invoice UI**: [http://localhost:3000](http://localhost:3000)

---

## 📦 Running Manual Containers (Docker CLI)

If you prefer to run specific services manually without Compose, use `docker run`.

### Run Admin Service (Monolith)

```bash
# Run Admin on port 8080
docker run -d \
  --name railzway-admin \
  -p 8080:8080 \
  -e DB_HOST=host.docker.internal \
  -e REDIS_HOST=host.docker.internal \
  ghcr.io/smallbiznis/railzway/railzway-admin:latest
```

### Run Scheduler (Background Jobs)

```bash
# Run Scheduler (No UI port needed)
docker run -d \
  --name railzway-scheduler \
  -e DB_HOST=host.docker.internal \
  -e REDIS_HOST=host.docker.internal \
  ghcr.io/smallbiznis/railzway/railzway-scheduler:latest
```

> **Note**: `host.docker.internal` allows the container to access services (like Postgres) running on your host machine. On Linux, use `--network host` or the host's actual IP address.

---

## ⚙️ Configuration (Environment Variables)

Common environment variables for `docker-compose` or `docker run`:

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | HTTP Port to listen on (Admin/Invoice). | `8080` |
| `DB_HOST` | Postgres Hostname. | `localhost` |
| `DB_USER` | Postgres Username. | `postgres` |
| `DB_PASSWORD` | Postgres Password. | (empty) |
| `REDIS_HOST` | Redis Hostname. | `localhost` |
| `API_URL` | (Invoice Service Only) URL to the Admin API. | `http://admin:8080` |
| `ENABLED_JOBS` | (Scheduler Only) Comma-separated list of jobs to run. | All jobs |

---

## 🏗️ Build Strategy (Advanced)

Railzway uses a **Unified Dockerfile** at the root. You don't usually need to build manually (Docker Compose handles it), but if you need to build specific images, use the `--target` flag:

```bash
# Admin Image
docker build --target railzway-admin -t railzway-admin .

# Invoice Image
docker build --target railzway-invoice -t railzway-invoice .

# Scheduler Image
docker build --target railzway-scheduler -t railzway-scheduler .

# Migration Image
docker build --target railzway-migration -t railzway-migration .
```
