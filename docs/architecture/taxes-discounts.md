# Taxes & Discounts

This document describes how Railzway currently computes taxes, invoice discounts, manual adjustments, and ledger-backed credits.

It is intentionally architecture-focused. It explains the financial role of each mechanism and where it appears in invoice construction.

## Current Mental Model

In Railzway, these concepts are related but not interchangeable:

- `tax rate` determines tax calculation
- `discount` reduces invoice subtotal during invoice construction
- `adjustment` is a manual invoice-side correction
- `credit` is ledger-backed value later applied against an open invoice

The current invoice flow is:

```text
gross subtotal
-> discount lines
-> net subtotal
-> tax lines
-> invoice total
-> ledger-backed credit use
-> payment collection
```

That order matters:

- discounts change the taxable base
- taxes are computed after discount application
- credits are not tax rules and are not coupon rules

## Tax Rates

A `TaxRate` defines the percentage and behavior of a tax applied to an invoice.

In the current implementation, tax rates are organization-scoped and loaded as active rates for the invoice organization.

### Inclusive vs. Exclusive Taxes

- **Exclusive Tax**: tax is added on top of the discounted subtotal.
  - Subtotal: $100
  - Tax (10%): $10
  - Total: $110
- **Inclusive Tax**: tax is already embedded in the discounted subtotal.
  - Price: $100
  - Tax (10%): $9.09
  - Total: $100

Railzway supports both inclusive and exclusive tax rates on the same invoice. Inclusive taxes are backed out from the discounted subtotal first, then exclusive taxes are added on top.

### Scope Today

Current tax scope is:

- **Organization level**: active `tax_rates` are attached by `org_id`

Not implemented today as first-class tax targeting:

- customer-specific tax rates
- plan-specific tax rates
- product-specific tax rules

Those may be added later, but they are not the current model.

## Invoice Discounts

Invoice discounts are no longer modeled as generic adjustments by default.

Railzway now supports coupon-driven and promotion-code-driven discounts as explicit invoice discount lines.

- a `coupon` defines the discount economics
- a `promotion code` provides the redemption handle
- invoice generation emits discount items when the coupon is eligible

For the full domain model, see [coupons-and-promotions.md](/Users/taufiktriantono/go/src/github.com/railzwaylabs/railzway/docs/architecture/coupons-and-promotions.md).

### Financial Role

Discounts:

- reduce the gross subtotal
- are computed before tax lines
- become auditable invoice items
- affect receivables and downstream payment collection

In current invoice generation, the engine:

1. builds gross invoice items
2. evaluates attached and auto-apply coupons
3. creates discount lines
4. computes net subtotal
5. computes tax lines from that net subtotal

So discounts are part of billing computation, not payment collection.

## Manual Invoice Adjustments

Railzway still supports generic invoice adjustments through `LineTypeAdjustment`.

These are not the same as coupon discounts.

Use cases for adjustments include:

- manual price corrections
- one-off service credits expressed directly on the invoice
- operator-entered non-coupon corrections

Adjustments are flexible, but they should not be used as a substitute for the coupon and promotion model when the business meaning is a reusable discount policy.

## Ledger-backed Credits

Railzway also supports ledger-backed customer credit balances.

This is separate from invoice discounts.

- **Credit Grant**: value is granted to the customer and recorded in the ledger as `SourceTypeCreditGrant`
- **Credit Use**: available balance is applied to an open invoice and recorded as `SourceTypeCreditUse`

Credits are useful for:

- goodwill balances
- refund carry-forward
- commercial credits
- manual account credits

The key distinction is:

- invoice discounts change invoice construction
- ledger credits settle part of an already-open receivable

That distinction should stay explicit in docs, APIs, and UI.

## Accuracy & Rounding

To preserve financial integrity:

- monetary values are stored in cents as integers
- percentage-based calculations are rounded to cents
- discount and tax calculations are applied in a deterministic order
- inclusive tax allocation is adjusted so the invoice remains internally consistent after rounding

This avoids penny drift and keeps invoice totals auditable.

> [!CAUTION]
> Once a tax rate has been used on an issued invoice, it should be marked `inactive` rather than edited or deleted. Historical invoices should continue to reflect the tax rule in force at the time they were issued.
