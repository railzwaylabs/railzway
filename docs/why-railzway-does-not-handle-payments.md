
# Why Railzway Does Not Handle Payments

Railzway intentionally does **not** handle payment execution.

This is not a missing feature.
It is a deliberate architectural decision.

Railzway determines **what should be billed**.
It does not determine **how money moves**.

---

## The Problem With Coupling Billing and Payments

In many systems, billing logic and payment execution are tightly coupled:

- pricing rules live inside payment flows
- billing state is inferred from payment results
- changes to pricing require changes to payment logic
- audits require reconstructing logic from side effects

This coupling creates several problems:

- billing behavior becomes implicit and hard to reason about
- historical correctness depends on mutable payment logic
- re-rating or price changes risk corrupting past invoices
- payment failures obscure billing intent

In short:

> **Money movement becomes the source of truth.**

This is fragile.

---

## Railzway’s Design Boundary

Railzway draws a strict boundary:

- **Billing** answers: *what should be billed, when, and why*
- **Payments** answer: *how money is collected or transferred*

Railzway owns the first.
It explicitly excludes the second.

This boundary allows Railzway to remain:

- deterministic
- auditable
- explainable
- payment-provider agnostic

---

## What Railzway Produces

Railzway produces **billing facts**, not financial side effects.

Examples:

- invoice line items
- billing states
- rated usage totals
- proration results
- billing cycle outcomes

These outputs are:

- derived solely from persisted inputs
- repeatable given the same configuration
- independent of payment success or failure

---

## What Railzway Does Not Do

Railzway does **not**:

- charge credit cards
- store payment methods
- retry failed payments
- manage settlements or payouts
- reconcile bank statements

These concerns belong to payment providers and financial systems.

---

## Why This Separation Matters

Separating billing from payments enables:

### Deterministic Billing

Billing results do not change because:

- a payment was retried
- a provider changed behavior
- a webhook arrived late

### Safe Pricing Changes

Pricing logic can evolve without:

- rewriting payment flows
- corrupting historical invoices
- breaking auditability

### Provider Flexibility

Teams can:

- integrate Stripe, Adyen, Midtrans, or others
- change providers without rewriting billing logic
- support multiple providers in parallel

### Clear Responsibility Boundaries

Failures are easier to reason about:

- billing bugs are billing bugs
- payment failures are payment failures

---

## Integration Model

Railzway is designed to sit **upstream** of payments.

A typical flow:

1. Application sends usage events to Railzway
2. Railzway computes billing state and invoices
3. Application passes invoice data to a payment provider
4. Payment provider executes collection
5. Payment results are optionally reflected back as metadata

Railzway remains the system of record for **billing intent**.

---

## Non-Goals

Railzway intentionally does not aim to become:

- a payment orchestration layer
- a merchant of record
- a financial ledger or accounting system
- a compliance abstraction over payment regulations

These domains have different constraints and responsibilities.

---

## Summary

Railzway does not handle payments because:

- billing logic must be deterministic
- money movement is inherently side-effectful
- coupling the two creates fragile systems
- clear boundaries improve long-term correctness

> **Railzway keeps billing boring, explicit, and predictable
> by refusing to own payment execution.**
>
