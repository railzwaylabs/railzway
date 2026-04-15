# Rate Limits (Public API)

Railzway applies layered limits on `/api/v1/usage-events` to protect system stability and keep ingestion predictable.

## What is limited

Limits apply in this order:

1. **Concurrency limit** per `customer_id + meter_code`
2. **Rate limit** per `subscription_id`
3. **Rate limit** per `customer_id`
4. **Rate limit** per `org`

When any limit is exceeded, the API responds with **HTTP 429**.

## Response headers

`/api/v1/usage-events` returns the following headers when limits are configured:

- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset` (seconds)
- `X-RateLimit-Scope` (`concurrency|subscription|customer|org`)
- `X-RateLimit-Reason` (`concurrency_limit|subscription_rate_limit|customer_rate_limit|org_rate_limit`)
- `Retry-After` (seconds, only on 429)

## Defaults

```
RATE_LIMIT_USAGE_EVENTS_WINDOW_SEC=60
RATE_LIMIT_USAGE_EVENTS_SUBSCRIPTION_PER_MIN=120
RATE_LIMIT_USAGE_EVENTS_CUSTOMER_PER_MIN=600
RATE_LIMIT_USAGE_EVENTS_ORG_PER_MIN=3000
RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_PER_CUSTOMER_METER=1
RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_TTL_SEC=5
```

## Tuning guidance

- Keep **subscription** limits lower than **customer** to prevent a single subscription from saturating the org.
- Use **concurrency=1** if you want deterministic, queue‑like ingestion.
- Increase **org** limits only when you can absorb peak bursts.

## Retry guidance

Use exponential backoff with jitter when receiving a 429 response.
