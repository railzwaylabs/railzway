#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
IMAGE_PREFIX="${IMAGE_PREFIX:-ghcr.io/railzwaylabs}"
TAG="${TAG:-latest}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
PUSH="${PUSH:-0}"

if [[ "$PUSH" == "1" ]]; then
  LOAD_FLAG="--push"
else
  # buildx can't --load multi-arch; default to amd64 when not pushing
  PLATFORMS="${PLATFORMS:-linux/amd64}"
  LOAD_FLAG="--load"
fi

cd "$ROOT_DIR"

echo "Building admin image..."
docker buildx build \
  --platform "$PLATFORMS" \
  -f Dockerfile \
  --target admin \
  -t "${IMAGE_PREFIX}/railzway-admin:${TAG}" \
  $LOAD_FLAG \
  .

echo "Building scheduler image..."
docker buildx build \
  --platform "$PLATFORMS" \
  -f Dockerfile \
  --target scheduler \
  -t "${IMAGE_PREFIX}/railzway-scheduler:${TAG}" \
  $LOAD_FLAG \
  .

echo "Building api image..."
docker buildx build \
  --platform "$PLATFORMS" \
  -f Dockerfile \
  --target api \
  -t "${IMAGE_PREFIX}/railzway-api:${TAG}" \
  $LOAD_FLAG \
  .

echo "Done."
