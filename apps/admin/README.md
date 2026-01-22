# Admin Service (`apps/admin`)

The **Admin Service** is the **Control Plane** of Railzway. It serves the internal dashboard used by operations teams to configure billing logic, view analytics, and manage customer accounts.

## Architecture

This service is a **hybrid** HTTP server:
1.  **Backend**: Go-based API (`/api/*`) for administration logic.
2.  **Frontend**: Embedded React/Vite SPA (served from root `/`).

## Features

- **Product Catalog**: Manage products, pricing, and features.
- **Customer Management**: View customer subscriptions and balances.
- **FinOps & Reporting**: Internal financial dashboards.
- **Configuration**: API keys, tax rules, and system settings.

## Running

### Development
You can run the frontend and backend separately for hot-reloading.

**Frontend:**
```bash
npm install
npm run dev
```

**Backend:**
```bash
export STATIC_DIR="./dist" # Point to build output if needed
go run main.go
```

### Production (Docker)
The Docker image bundles both:
```bash
docker run -p 8080:8080 ghcr.io/<org>/railzway-admin:latest
```
