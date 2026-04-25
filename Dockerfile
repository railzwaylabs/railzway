# syntax=docker/dockerfile:1.7

FROM node:20-alpine AS admin-ui
WORKDIR /app
RUN corepack enable

# Install deps with maximum cache reuse
COPY pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/admin/package.json apps/admin/package.json
COPY packages/invoice-ui/package.json packages/invoice-ui/package.json
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
  pnpm --dir apps/admin install --frozen-lockfile

# Build admin UI
COPY apps/admin apps/admin
COPY packages/invoice-ui packages/invoice-ui
COPY packages/design-system packages/design-system
RUN --mount=type=cache,target=/root/.cache \
  pnpm --dir apps/admin build

FROM golang:1.25.1-alpine AS go-base
WORKDIR /app
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum go.work go.work.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY cmd cmd
COPY internal internal
COPY config config

FROM go-base AS build-admin
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
  go build -trimpath -ldflags="-s -w" -o /out/admin ./cmd/admin

FROM go-base AS build-scheduler
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
  go build -trimpath -ldflags="-s -w" -o /out/scheduler ./cmd/scheduler

FROM go-base AS build-api
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
  go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM go-base AS build-migrate
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
  go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM scratch AS migrate-assets
COPY db /db

FROM gcr.io/distroless/static:nonroot AS admin-api
WORKDIR /app
COPY --from=build-admin /out/admin /app/admin
COPY --from=go-base /app/config /app/config
COPY features.yml /app/features.yml
EXPOSE 8080
ENTRYPOINT ["/app/admin"]

FROM gcr.io/distroless/static:nonroot AS admin-all-in-one
WORKDIR /app
COPY --from=build-admin /out/admin /app/admin
COPY --from=admin-ui /app/apps/admin/dist /app/apps/admin/dist
COPY --from=go-base /app/config /app/config
COPY features.yml /app/features.yml
EXPOSE 8080
ENTRYPOINT ["/app/admin"]

FROM gcr.io/distroless/static:nonroot AS scheduler
WORKDIR /app
COPY --from=build-scheduler /out/scheduler /app/scheduler
COPY --from=go-base /app/config /app/config
COPY features.yml /app/features.yml
ENTRYPOINT ["/app/scheduler"]

FROM gcr.io/distroless/static:nonroot AS api
WORKDIR /app
COPY --from=build-api /out/api /app/api
COPY --from=go-base /app/config /app/config
COPY features.yml /app/features.yml
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM gcr.io/distroless/static:nonroot AS migrate
WORKDIR /app
COPY --from=build-migrate /out/migrate /app/railzway-migrate
COPY --from=go-base /app/config /app/config
COPY --from=migrate-assets /db /app/db
ENTRYPOINT ["/app/railzway-migrate"]
