# Coupons and Segments API

This document describes the admin API contract for coupons, promotion codes, billing segments, and automatic invoice discounts.

## Scope

Coupons are reusable discount rules. Promotion codes are redeemable codes that attach a coupon to a subscription. Billing segments are organization-scoped keys used to target coupons without relying on free-text UI input.

The current implementation supports:

- Percent and fixed-amount coupons.
- Manual redemption through promotion codes.
- Subscription-level coupon attachment.
- Auto-apply coupons during invoice generation.
- Date-based coupon validity windows.
- Coupon targeting by billing segment.
- Discount invoice items with auditable metadata.

## Data Model

### Coupon

Relevant fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | UUID | Coupon ID. |
| `org_id` | UUID | Tenant organization. |
| `name` | string | Display name. |
| `type` | string | `PERCENT` or `FIXED`. |
| `percentage` | number | Required for percent coupons. Must be `> 0` and `<= 100`. |
| `amount_cents` | integer | Required for fixed coupons. Must be `> 0`. |
| `currency` | string | Required for fixed coupons. Three-letter currency code. |
| `duration` | string | `ONCE`, `REPEATING`, or `FOREVER`. |
| `duration_months` | integer | Required when `duration = REPEATING`. |
| `valid_from` | RFC3339 timestamp | Optional lower bound for invoice period eligibility. |
| `valid_until` | RFC3339 timestamp | Optional upper bound for invoice period eligibility. |
| `auto_apply` | boolean | If true, invoice generation can apply the coupon without a promotion code. |
| `target_segment` | string | Optional billing segment key. Empty means all segments. |

### Billing Segment

Relevant fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | UUID | Segment ID. |
| `org_id` | UUID | Tenant organization. |
| `key` | string | Stable machine key, for example `startup`. |
| `name` | string | Display name. |
| `scope` | string | `customer`, `subscription`, or `any`. |
| `description` | string | Optional description. |
| `active` | boolean | Inactive segments cannot be used for new coupon targeting. |

Segment keys are normalized to lowercase and may contain letters, numbers, `_`, and `-`.

Default segments are inserted idempotently by the coupon service when segments are listed or a coupon target segment is validated:

- `startup`
- `enterprise`
- `education`
- `partner`
- `internal`
- `self_serve`
- `sales_led`
- `annual_contract`
- `trial_converted`
- `reactivated`
- `loyal_customer`
- `high_spend`
- `committed_spend`

## Coupon Endpoints

All endpoints require admin authentication, CSRF protection, and organization context.

### Create Coupon

```http
POST /admin/v1/coupons
```

Percent coupon:

```json
{
  "name": "Launch 20%",
  "type": "PERCENT",
  "percentage": 20,
  "duration": "REPEATING",
  "duration_months": 6,
  "valid_from": "2026-05-01T00:00:00Z",
  "valid_until": "2026-12-01T00:00:00Z",
  "auto_apply": true,
  "target_segment": "startup"
}
```

Fixed-amount coupon:

```json
{
  "name": "USD 10 off",
  "type": "FIXED",
  "amount_cents": 1000,
  "currency": "USD",
  "duration": "ONCE",
  "auto_apply": false
}
```

Response:

```json
{
  "coupon": {
    "id": "8f92e17a-0704-4a8c-88a8-241c1e77a3f3",
    "org_id": "3c4c4cb0-b53e-4e81-98a6-598416d8dbac",
    "name": "Launch 20%",
    "type": "PERCENT",
    "percentage": 20,
    "duration": "REPEATING",
    "duration_months": 6,
    "valid_from": "2026-05-01T00:00:00Z",
    "valid_until": "2026-12-01T00:00:00Z",
    "auto_apply": true,
    "target_segment": "startup",
    "created_at": "2026-04-23T00:00:00Z",
    "updated_at": "2026-04-23T00:00:00Z"
  }
}
```

Validation:

- `name` is required.
- `type` must be `PERCENT` or `FIXED`.
- `PERCENT` requires `percentage > 0 && percentage <= 100`.
- `FIXED` requires `amount_cents > 0` and a three-letter `currency`.
- `duration` must be `ONCE`, `REPEATING`, or `FOREVER`.
- `REPEATING` requires `duration_months > 0`.
- If both are provided, `valid_until` must be after `valid_from`.
- `target_segment`, when provided, must exist and be active in `billing_segments`.

### List Coupons

```http
GET /admin/v1/coupons
```

Response:

```json
{
  "coupons": [
    {
      "id": "8f92e17a-0704-4a8c-88a8-241c1e77a3f3",
      "name": "Launch 20%",
      "type": "PERCENT",
      "percentage": 20,
      "duration": "REPEATING",
      "duration_months": 6,
      "auto_apply": true,
      "target_segment": "startup"
    }
  ]
}
```

## Promotion Code Endpoints

### Create Promotion Code

```http
POST /admin/v1/promotion-codes
```

Request:

```json
{
  "coupon_id": "8f92e17a-0704-4a8c-88a8-241c1e77a3f3",
  "code": "LAUNCH20",
  "active": true,
  "max_redemptions": 500
}
```

Response:

```json
{
  "promotion_code": {
    "id": "ab87f7cc-9c69-4f3d-9dd8-67e5e799a291",
    "coupon_id": "8f92e17a-0704-4a8c-88a8-241c1e77a3f3",
    "code": "LAUNCH20",
    "active": true,
    "max_redemptions": 500,
    "redemption_count": 0
  }
}
```

Promotion codes are normalized to uppercase before storage and lookup.

### List Promotion Codes

```http
GET /admin/v1/promotion-codes
```

Response:

```json
{
  "promotion_codes": [
    {
      "id": "ab87f7cc-9c69-4f3d-9dd8-67e5e799a291",
      "coupon_id": "8f92e17a-0704-4a8c-88a8-241c1e77a3f3",
      "code": "LAUNCH20",
      "active": true,
      "max_redemptions": 500,
      "redemption_count": 0
    }
  ]
}
```

### Redeem Promotion Code

```http
POST /admin/v1/subscriptions/{subscription_id}/promotion-code
```

Request:

```json
{
  "code": "LAUNCH20"
}
```

Response:

```json
{
  "coupon": {
    "id": "8f92e17a-0704-4a8c-88a8-241c1e77a3f3",
    "name": "Launch 20%",
    "type": "PERCENT",
    "percentage": 20
  }
}
```

Semantics:

- A subscription can have one attached coupon.
- Redeeming another valid promotion code replaces the previous subscription coupon.
- `redemption_count` increments only after a successful redemption.
- `promotion_code_inactive` is returned for inactive codes.
- `promotion_code_max_redemptions_reached` is returned when the redemption cap is exhausted.

## Segment Endpoints

### Create Segment

```http
POST /admin/v1/segments
```

Request:

```json
{
  "key": "startup",
  "name": "Startup",
  "scope": "customer",
  "description": "Early-stage or venture-backed customer",
  "active": true
}
```

Response:

```json
{
  "segment": {
    "id": "0cb628fb-bc73-4b6d-8f7a-69ca8d4ff847",
    "org_id": "3c4c4cb0-b53e-4e81-98a6-598416d8dbac",
    "key": "startup",
    "name": "Startup",
    "scope": "customer",
    "description": "Early-stage or venture-backed customer",
    "active": true,
    "created_at": "2026-04-23T00:00:00Z",
    "updated_at": "2026-04-23T00:00:00Z"
  }
}
```

Validation:

- `key` is required and normalized to lowercase.
- `key` must be unique per organization.
- `name` is required.
- `scope` must be `customer`, `subscription`, or `any`. Empty scope defaults to `customer`.

### List Segments

```http
GET /admin/v1/segments
```

Query parameters:

| Parameter | Type | Notes |
| --- | --- | --- |
| `scope` | string | Optional. `customer`, `subscription`, or `any`. If omitted, all scopes are returned. |
| `include_inactive` | boolean | Optional. Defaults to false. |

Response:

```json
{
  "segments": [
    {
      "id": "0cb628fb-bc73-4b6d-8f7a-69ca8d4ff847",
      "key": "startup",
      "name": "Startup",
      "scope": "customer",
      "active": true
    }
  ]
}
```

When `scope=customer`, the API returns `customer` and `any` segments. When `scope=subscription`, it returns `subscription` and `any` segments.

### Update Segment

```http
PATCH /admin/v1/segments/{segment_key}
```

Request:

```json
{
  "name": "Startup Program",
  "scope": "customer",
  "description": "Customers enrolled in the startup program",
  "active": true
}
```

Response:

```json
{
  "segment": {
    "id": "0cb628fb-bc73-4b6d-8f7a-69ca8d4ff847",
    "key": "startup",
    "name": "Startup Program",
    "scope": "customer",
    "description": "Customers enrolled in the startup program",
    "active": true
  }
}
```

The segment key is immutable. Create a new segment if the key needs to change.

## Invoice Discount Behavior

Invoice generation evaluates discounts after charge lines are built and before taxes are calculated.

Discount candidates include:

1. The coupon attached to the subscription through promotion-code redemption.
2. Active `auto_apply` coupons whose validity window intersects the invoice period.

Eligibility checks:

- The coupon type and monetary fields must be valid.
- The coupon validity window must intersect the invoice period.
- Fixed coupons must match the subscription currency.
- If `target_segment` is set, it must match the customer or subscription segment.

Current segment lookup checks:

1. `customers.metadata->>'segment'`
2. `subscriptions.metadata->>'segment'`

Customer metadata wins when both are present.

Discounts are added as `invoice_items` with `line_type = "discount"`. The discount item amount is positive, and invoice totals subtract the discount internally.

Example discount item metadata:

```json
{
  "coupon_id": "8f92e17a-0704-4a8c-88a8-241c1e77a3f3",
  "coupon_type": "PERCENT",
  "applied_at": "2026-05-01T00:00:00Z",
  "source": "auto_apply"
}
```

## Duration Semantics

`ONCE` applies to the invoice period that contains the coupon applied time. For auto-apply coupons, the applied time is `valid_from`, then `created_at`, then the invoice period start as fallback.

`REPEATING` applies from the coupon applied time through `duration_months`.

`FOREVER` applies while the validity window permits it.

`valid_from` and `valid_until` constrain all duration modes.

## Error Codes

| Code | Meaning |
| --- | --- |
| `invalid_coupon` | Coupon payload or eligibility field is invalid. |
| `invalid_promotion_code` | Promotion code payload is invalid. |
| `promotion_code_inactive` | Promotion code exists but is inactive. |
| `promotion_code_max_redemptions_reached` | Promotion code redemption limit has been reached. |
| `invalid_segment` | Segment payload is invalid, missing, or inactive. |
| `segment_exists` | Segment key already exists for the organization. |
| `not_found` | Requested coupon, promotion code, segment, or subscription was not found. |

## Current Limitations

- Coupon targeting supports one `target_segment` per coupon.
- Customer/subscription segment assignment is still read from metadata, not from a dedicated assignment table.
- Product, plan, price, meter, subscription age, first invoice, minimum spend, priority, and stackability rules are not yet implemented.
- Auto-apply coupons are applied sequentially by creation time and cannot reduce the invoice subtotal below zero.

## Future Direction

The next eligibility fields should be modeled separately from `target_segment`:

- `target_product_id`
- `target_plan_id`
- `target_price_id`
- `target_meter_id`
- `minimum_subtotal_cents`
- `subscription_age_months_gte`
- `first_invoice_only`
- `stackable`
- `priority`

Keeping these as explicit eligibility rules avoids overloading segment keys with dynamic billing conditions.
