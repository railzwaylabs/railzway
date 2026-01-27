# syntax=docker/dockerfile:1

# ==========================================
# Stage 1: Backend Builder (Shared)
# ==========================================
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend-builder

# Platform arguments for cross-compilation
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Build arguments
ARG VERSION="1.0.0"
ARG CLOUD_METRICS_ENDPOINT=""
ARG CLOUD_METRICS_AUTH_TOKEN=""

# Cached Dependency Download
COPY go.mod go.sum ./
RUN go mod download

# Build Source
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags \
    "-X github.com/smallbiznis/railzway/internal/config.DefaultAppVersion=${VERSION} \
     -X github.com/smallbiznis/railzway/internal/config.DefaultCloudMetricsEndpoint=${CLOUD_METRICS_ENDPOINT} \
     -X github.com/smallbiznis/railzway/internal/config.DefaultCloudMetricsAuthToken=${CLOUD_METRICS_AUTH_TOKEN}" \
    -o railzway ./cmd/railzway

# ==========================================
# Stage 2: Admin Frontend Builder
# ==========================================
FROM node:20-alpine AS admin-builder
WORKDIR /app
COPY apps/admin/package.json apps/admin/pnpm-lock.yaml ./
RUN npm install -g pnpm
RUN pnpm install --frozen-lockfile
COPY apps/admin/ ./
RUN pnpm run build

# ==========================================
# Stage 3: Invoice Frontend Builder
# ==========================================
FROM node:20-alpine AS invoice-builder
WORKDIR /app
COPY apps/invoice/package.json apps/invoice/pnpm-lock.yaml ./
RUN npm install -g pnpm
RUN pnpm install --frozen-lockfile
COPY apps/invoice/ ./
RUN pnpm run build

# ==========================================
# Target: Railzway Admin
# ==========================================
FROM alpine:latest AS railzway-admin
WORKDIR /app
COPY --from=backend-builder /app/railzway .
COPY --from=admin-builder /app/dist ./public
RUN mkdir -p /var/lib/railzway/config
ENV PORT=8080
ENV STATIC_DIR=./public
ENV GIN_MODE=release
VOLUME ["/var/lib/railzway"]
EXPOSE 8080
CMD ["./railzway", "serve"]

# ==========================================
# Target: Railzway Invoice
# ==========================================
FROM alpine:latest AS railzway-invoice
WORKDIR /app
COPY --from=backend-builder /app/railzway .
COPY --from=invoice-builder /app/dist ./public
RUN mkdir -p /var/lib/railzway/config
ENV PORT=8080
ENV STATIC_DIR=./public
ENV GIN_MODE=release
VOLUME ["/var/lib/railzway"]
EXPOSE 8080
CMD ["./railzway", "serve"]

# ==========================================
# Target: Railzway Scheduler
# ==========================================
FROM alpine:latest AS railzway-scheduler
WORKDIR /app
COPY --from=backend-builder /app/railzway .
RUN mkdir -p /var/lib/railzway/config
ENV PORT=8080
ENV GIN_MODE=release
VOLUME ["/var/lib/railzway"]
EXPOSE 8080
CMD ["./railzway", "scheduler"]

# ==========================================
# Target: Railzway Migration
# ==========================================
FROM alpine:latest AS railzway-migration
WORKDIR /app
COPY --from=backend-builder /app/railzway .
RUN mkdir -p /var/lib/railzway/config
ENV GIN_MODE=release
VOLUME ["/var/lib/railzway"]
CMD ["./railzway", "migrate"]
