# Billing Automation API

This document describes the v1.1.0 billing automation contract for coupons, ledger-backed credits, invoice generation, and reconciliation integrity checks.

## Scope

The current backend implementation introduces:

- Coupon and promotion-code persistence through `coupons`, `promotion_codes`, and `subscription_coupons`.
- Invoice discount calculation during draft invoice generation and recalculation.
- Ledger-backed customer credit draw when a draft invoice is opened.
- Reconciliation checks for coupon mismatch and ledger credit mismatch.

Coupon route handlers are not yet exposed in the admin HTTP layer. The endpoint contracts below define the API shape that should be wired to the coupon service before marking the Coupon Engine as externally available.

## Monetary Rules

All monetary values are integer cents. Percent coupons are rounded to the nearest cent at the invoice-line level.

The invoice calculation order is:

1. Build subscription and usage charge lines.
2. Apply the active subscription coupon as a positive `discount` line.
3. Calculate tax on the discounted subtotal.
4. Compute `total_cents = discounted_subtotal + exclusive_tax_cents`.
5. On invoice open, draw available ledger credits after tax and reduce `amount_due_cents`.

Credits are treated as a payment method, not as a taxable discount.

## Coupons

### Create Coupon

Planned admin endpoint:

```http
POST /admin/v1/coupons
```

Request:

```json
{
  "name": "Launch 20%",
  "type": "PERCENT",
  "percentage": 20,
  "duration": "REPEATING",
  "duration_months": 6
}
```

Fixed-amount coupon request:

```json
{
  "name": "USD 10 off",
  "type": "FIXED",
  "amount_cents": 1000,
  "currency": "USD",
  "duration": "ONCE"
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
    "created_at": "2026-04-20T06:00:00Z",
    "updated_at": "2026-04-20T06:00:00Z"
  }
}
```

Validation:

- `type` must be `PERCENT` or `FIXED`.
- `PERCENT` requires `percentage > 0 && percentage <= 100`.
- `FIXED` requires `amount_cents > 0` and a 3-letter `currency`.
- `duration` must be `ONCE`, `REPEATING`, or `FOREVER`.
- `REPEATING` requires `duration_months > 0`.

### List Coupons

Planned admin endpoint:

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
      "duration_months": 6
    }
  ]
}
```

### Create Promotion Code

Planned admin endpoint:

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

### Redeem Promotion Code

Planned checkout/admin endpoint:

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
    "percentage": 20,
    "duration": "REPEATING",
    "duration_months": 6
  }
}
```

Semantics:

- A subscription can have one active coupon attachment.
- Redeeming another valid code replaces the previous subscription coupon.
- `redemption_count` increments when redemption succeeds.
- `promotion_code_inactive` is returned for inactive codes.
- `promotion_code_max_redemptions_reached` is returned when the configured cap is exhausted.

## Invoice Generation

Existing admin endpoint:

```http
POST /admin/v1/invoices/generate
```

When a subscription has an applicable coupon, invoice generation adds a `discount` invoice item.

Example response fragment:

```json
{
  "invoice": {
    "subtotal_cents": 8000,
    "tax_cents": 800,
    "total_cents": 8800,
    "amount_due_cents": 8800,
    "amount_paid_cents": 0
  },
  "items": [
    {
      "line_type": "subscription",
      "amount_cents": 10000
    },
    {
      "line_type": "discount",
      "description": "Coupon discount: Launch 20%",
      "amount_cents": 2000,
      "metadata": {
        "coupon_id": "8f92e17a-0704-4a8c-88a8-241c1e77a3f3",
        "coupon_type": "PERCENT"
      }
    },
    {
      "line_type": "tax",
      "amount_cents": 800
    }
  ]
}
```

The `discount` line amount is positive. Consumers must not sum every line as additive revenue. Use invoice-level totals for financial decisions.

## Ledger Credits

Customer credits use the ledger account code `credits`. A credit grant should be represented as a balanced ledger transaction that credits `credits` for the customer.

Example credit grant:

```http
POST /admin/v1/ledger/transactions
```

```json
{
  "currency": "USD",
  "source_type": "credit_grant",
  "source_id": "6ff50dc7-843a-4a62-8797-066bd3b64020",
  "customer_id": "ebef60c7-7117-466b-a33d-0555f104b47f",
  "idempotency_key": "credit_grant:ebef60c7-7117-466b-a33d-0555f104b47f:2026-04-20",
  "entries": [
    {
      "account_code": "1000_cash",
      "entry_type": "debit",
      "amount_cents": 5000,
      "currency": "USD"
    },
    {
      "account_code": "credits",
      "entry_type": "credit",
      "amount_cents": 5000,
      "currency": "USD"
    }
  ]
}
```

When an invoice is opened, the engine:

- Reads the customer balance from account `credits`.
- Applies up to `amount_due_cents`.
- Adds an `adjustment` invoice item with metadata `source = ledger_credit_draw`.
- Posts a `credit_use` ledger transaction that debits `credits` and credits `1100_accounts_receivable`.
- Sets `status = paid` if the credit fully covers the invoice.

Example opened invoice fragment after a partial credit draw:

```json
{
  "invoice": {
    "status": "open",
    "total_cents": 8800,
    "amount_paid_cents": 3000,
    "amount_due_cents": 5800
  },
  "items": [
    {
      "line_type": "adjustment",
      "description": "Customer credit applied",
      "amount_cents": 3000,
      "metadata": {
        "source": "ledger_credit_draw",
        "account_code": "credits"
      }
    }
  ]
}
```

## Reconciliation

The reconciliation scheduler records audit events when integrity checks fail.

Audit actions:

- `reconciliation.coupon_mismatch`: the invoice discount line does not match the subscription coupon rule.
- `reconciliation.ledger_credit_mismatch`: the invoice credit adjustment does not match a debit in ledger account `credits`.
- `reconciliation.usage_mismatch`: invoice usage total differs from rating results.
- `reconciliation.ledger_mismatch`: billing-cycle ledger credits differ from invoice total.

Example audit metadata for coupon mismatch:

```json
{
  "coupon_id": "8f92e17a-0704-4a8c-88a8-241c1e77a3f3",
  "expected_discount_cents": 2000,
  "invoice_discount_cents": 1500
}
```

Example audit metadata for ledger credit mismatch:

```json
{
  "invoice_credit_adjustment_cents": 3000,
  "ledger_credit_debit_cents": 0
}
```

## Error Codes

| Code | Meaning |
| --- | --- |
| `invalid_coupon` | Coupon payload or attachment is invalid. |
| `invalid_promotion_code` | Promotion code payload is invalid. |
| `promotion_code_inactive` | Promotion code exists but is inactive. |
| `promotion_code_max_redemptions_reached` | Promotion code has reached its redemption cap. |
| `not_found` | Coupon, promotion code, subscription, or invoice was not found in the organization context. |
| `invalid_organization` | Request does not have a valid organization context. |

## Idempotency

Use deterministic idempotency keys for financial mutations:

- Invoice generation: `invoice_generate:{subscription_id}:{period_start}:{period_end}`.
- Credit grants: `credit_grant:{customer_id}:{external_reference}`.
- Payment collection: `invoice_pay:{invoice_id}`.

The automatic credit draw uses the internal idempotency key `invoice_credit_draw:{invoice_id}`.
