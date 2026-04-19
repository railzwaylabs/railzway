# Retry Policy (Client FE + Partner API)

This project standardizes a conservative retry policy to avoid double-billing while still handling transient failures.

## FE (Admin UI)
- **Retries**: only for `GET/HEAD/OPTIONS`.
- **No automatic retries** for `POST/PUT/PATCH/DELETE` to avoid duplicates.
- **Retryable status codes**: `429`, `500`, `502`, `503`, `504`.
- **Backoff**: exponential with cap.
  - Base delay: `250ms`
  - Max delay: `2000ms`
  - Max attempts: `3` (1 initial + 2 retries)
- **Retry-After** honored** when present.

Rationale: Admin UI should never auto-repeat a write unless the call is explicitly idempotent.

## Partner API
- **Idempotency is required** for all write endpoints (create/update/generate).
- Clients **must** send `idempotency_key` and **reuse the same value** on retry.
- Retry only on transient failures:
  - Network timeout / connection reset
  - `429`, `500`, `502`, `503`, `504`
- Recommended schedule:
  - 1st retry: `250ms`
  - 2nd retry: `500ms`
  - 3rd retry: `1000ms`
  - Stop after **3 retries** (total 4 attempts)
  - Respect `Retry-After` if provided

## Notes
- If the client receives **timeout after request sent**, it **must retry with same idempotency key**.
- For bulk ingestion, use **batch idempotency keys** or per-event keys.
- Retries must not bypass rate limits; use `Retry-After`.
