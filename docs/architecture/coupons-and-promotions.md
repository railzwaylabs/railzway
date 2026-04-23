# Coupons and Promotions

This document describes the billing-domain model for coupons, promotion codes, billing segments, and automatic invoice discounts in Railzway.

It is intentionally architecture-focused. It explains what the concepts mean, how they relate to each other, and where they fit into invoice generation. It does not try to be an endpoint catalog.

## Purpose

Railzway separates discount policy from discount redemption.

- A `coupon` defines the discount rule.
- A `promotion code` provides a redeemable handle for that rule.
- A `billing segment` provides an organization-scoped targeting key for automatic coupon eligibility.

This separation allows Railzway to support both:

- operator-driven automatic discounts
- user- or subscription-driven coupon redemption

## Core Entities

### Coupon

A coupon is the canonical discount rule.

Current attributes include:

- discount type:
  - `PERCENT`
  - `FIXED`
- duration:
  - `ONCE`
  - `REPEATING`
  - `FOREVER`
- optional validity window:
  - `valid_from`
  - `valid_until`
- optional automatic application:
  - `auto_apply`
- optional target segment:
  - `target_segment`

A coupon answers:

> if this discount is eligible, how should the invoice amount be reduced?

### Promotion Code

A promotion code is the redeemable code attached to a coupon.

Examples:

- `WELCOME20`
- `STARTUP2026`
- `PARTNER-BETA`

A promotion code answers:

> how does a user or operator reference a coupon during redemption?

Important distinction:

- coupon = discount rule
- promotion code = redemption handle for that rule

One coupon may have multiple promotion codes.

### Billing Segment

A billing segment is an organization-scoped key used to target discounts without relying on free-text input.

Examples:

- `startup`
- `enterprise`
- `annual_contract`
- `trial_converted`

Segments let operators say:

> this coupon should only be considered for subscriptions or customers in a particular commercial cohort

This keeps coupon targeting explicit and avoids fragile string matching in the UI.

## Coupon vs Promotion Code

This distinction is important.

### Coupon

Represents the economics of the discount:

- percentage or fixed amount
- duration
- date validity
- auto-apply behavior
- segment targeting

### Promotion Code

Represents the redemption mechanism:

- code string
- active/inactive state
- max redemptions
- redemption count

In practice:

- an operator may create one coupon
- then create multiple promotion codes that all point to it

This supports different acquisition channels without duplicating discount logic.

## Application Modes

Railzway currently supports two discount application modes.

### 1. Promotion-code-driven application

Flow:

1. a promotion code is redeemed for a subscription
2. the underlying coupon becomes the subscription coupon
3. invoice generation evaluates that attached coupon
4. a discount line is emitted if the coupon is eligible for the invoice period

This is the classic redeemable-code path.

### 2. Auto-apply coupon evaluation

Flow:

1. invoice generation starts
2. the system evaluates coupons with `auto_apply = true`
3. validity window and segment targeting are checked
4. eligible coupons are converted into invoice discount lines

This is the operator-managed automatic discount path.

It is closer to how cloud vendors apply recurring credits or promotions without requiring end users to enter a code manually.

## Invoice Lifecycle Integration

Coupons and promotion codes do not act directly on payments. They act on invoice construction.

The current mental model is:

```text
usage events
-> metering / aggregation
-> rating
-> invoice line construction
-> draft invoice
-> open invoice
-> ledger posting
-> payment adapter / provider
```

Coupons and promotion codes affect the `invoice line construction` stage.

More precisely:

- subscription charges are assembled
- usage charges are assembled
- coupon eligibility is evaluated
- discount lines are created
- taxes and totals are computed

So the discount model sits inside billing computation, not inside payment collection.

## Invoice Discount Representation

When a coupon is applied, Railzway emits a discount invoice item with auditable metadata.

That metadata should make it possible to answer:

- which coupon was applied?
- was it auto-applied or subscription-attached?
- which rule matched?
- when did it expire?

This is important because discounts affect:

- invoice totals
- receivables
- revenue recognition inputs
- downstream payment collection

Discounts must therefore be traceable, not just visually present in UI.

## Segment Targeting

Segment targeting is intentionally simple today.

A coupon may optionally specify `target_segment`.

That means:

- if empty, the coupon is not segment-restricted
- if present, the invoice/subscription/customer must match that segment to be eligible

Current segment usage is best understood as commercial targeting, not a full rule engine.

Examples:

- startup launch credits
- partner-only promotions
- annual contract incentives
- trial conversion offers

## Current Constraints

Current implementation constraints include:

- segment targeting is still relatively simple
- coupon eligibility is not yet a full rule engine
- plan-, product-, price-, and meter-level targeting are future enhancements
- stacking policy is limited
- promotion codes are tied to coupon redemption, not a broader campaign system

So the current model is strong enough for:

- fixed or percentage discounts
- manual redemption
- auto-apply promotions
- segment-based targeting

But it is not yet a complete enterprise promotion engine.

## Future Direction

The natural next steps are:

- richer eligibility rules:
  - product-level targeting
  - plan-level targeting
  - price-level targeting
  - meter-level targeting
- first-invoice-only or minimum-spend rules
- clearer stacking and priority rules
- customer- or contract-level overrides
- better release/audit visibility for discount sources

Those changes should extend the current model, not replace it.

The key principle should stay the same:

> coupons define discount economics, promotion codes define redemption, and invoice generation remains the place where discount effects become financial truth.
