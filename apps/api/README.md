# API Service (`apps/api`)

The **API Service** is the **Data Plane** of Railzway. It handles high-volume, programmatic access to the billing engine.

## Purpose

- **Usage Ingestion**: `POST /api/usage` for reporting metering events.
- **Data Querying**: Fetching invoices, customers, and subscriptions via API.
- **Integration**: Webhooks (future) and machine-to-machine communication.

This service is optimized for **throughput** and **low latency**. It does not serve any UI assets.

## Running

```bash
# Run locally
go run ./apps/api/main.go

# Docker
docker run -p 8080:8080 ghcr.io/<org>/railzway-api:latest
```

## Configuration

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | HTTP listen port. |
| `DATABASE_URL` | - | Postgres connection string. |
| `GIN_MODE` | `release` | Set to `debug` for verbose logging. |
