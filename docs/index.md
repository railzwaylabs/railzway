# The Railzway Billing Model

Railzway is built on a simple premise:

> **Billing is a computation problem, not a payment problem.**

This documentation describes the mental model behind Railzway’s billing engine—
the principles, boundaries, and design decisions that shape how billing is
computed, explained, and corrected.

---

## What Railzway Optimizes For

Railzway optimizes for:

- correctness over immediacy
- determinism over convenience
- explicit intent over inferred behavior
- long-term auditability over short-term flexibility

These priorities inform every design decision in the system.

---

## Core Concepts

Railzway’s billing model is composed of a small set of explicit concepts.

Each concept is documented independently, but designed to work together.

---

### Billing vs Payments

Billing determines **what should be billed**.
Payments determine **how money moves**.

Railzway owns billing and **orchestrates** payments.
It delegates actual processing to specialized providers.

- [Why Railzway Does Not Handle Payments](why-railzway-does-not-handle-payments.md)

---

### Deterministic Computation

Billing results must be reproducible.

Given the same inputs, Railzway will always produce the same outputs.
No result depends on runtime side effects or external systems.

- [How Billing Stays Deterministic](how-billing-stays-deterministic.md)

---

### Pricing Over Time

Pricing changes must not rewrite history.

Railzway models pricing as versioned, append-only configuration
with explicit effective dates.

- [Pricing Versioning and Effective Dates](pricing-versioning-and-effective-dates.md)

---

### Append-Only Billing Data

Billing data represents financial intent.

Once recorded, intent is never mutated.
Corrections are modeled explicitly.

- [Why Billing Is Append-Only](why-billing-is-append-only.md)

---

### Billing Cycles

Billing cycles define computation boundaries.

Each cycle owns:

- a fixed time window
- a stable set of inputs
- a deterministic output

- [Billing Cycle as a First-Class Concept](billing-cycle-as-a-first-class-concept.md)

---

### Late-Arriving Usage

Usage may arrive late.

Railzway evaluates usage by event time,
and handles late data explicitly without mutating history.

- [Handling Late-Arriving Usage](handling-late-arriving-usage.md)

---

### Idempotent Usage Ingestion

Usage ingestion must survive retries.

Railzway enforces idempotency to ensure that
retries never affect billing correctness.

- [Idempotency and Usage Ingestion](idempotency-and-usage-ingestion.md)

---

### Non-Real-Time Billing

Billing is not computed in real time.

Usage is ingested continuously.
Billing is computed once the context is complete.

- [Why Billing Is Not Real-Time](why-billing-is-not-real-time.md)

---

### Explainability and Observability

Billing results must be explainable.

Every invoice can be traced back to:

- usage events
- pricing versions
- billing cycles
- configuration state

- [Observability and Billing Explainability](observability-and-billing-explainability.md)

---

### Corrections and Adjustments

Mistakes are inevitable.
Silent mutation is not acceptable.

Railzway models corrections as explicit adjustments
that preserve historical integrity.

- [Billing Corrections and Adjustments](billing-corrections-and-adjustments.md)

---

## API & Operations

Technical guides for integrating and operating Railzway.

- [API Versioning Strategy](API_VERSIONING.md)
- [Pagination Guide](PAGINATION.md)
- [Error Taxonomy](ERROR_TAXONOMY.md)
- [Quality Assessment](QUALITY_ASSESSMENT.md)
- [Swagger Documentation](swagger.yaml)

---

## Architectural Decision Records (ADRs)

Key architectural decisions and their rationale.

- [ADR-001: Billing Truth & Source of Record](adr/ADR-001-billing-truth-and-source-of-record.md)
- [ADR-002: Why usage_events.status.rated Is Deferred](adr/ADR-002-usage-status-rated-deferred.md)
- [ADR-003: API Versioning Strategy](adr/ADR-003-api-versioning-strategy.md)

---

## How to Read This Documentation

This documentation is not a tutorial.

It is a set of design decisions that define how Railzway works.
Readers are encouraged to read documents in any order,
but understanding emerges from their combination.

---

## What This Model Enables

By enforcing these principles, Railzway enables:

- predictable billing behavior
- safe pricing evolution
- reliable audits
- clear separation of concerns
- explainable financial outcomes

These properties are difficult to retrofit.
Railzway makes them foundational.

---

## Closing

Railzway’s billing model is intentionally conservative.

It prioritizes correctness, clarity, and trust over speed and novelty.

> **Billing should be boring, explicit, and explainable.
> Railzway exists to make that possible.**
>
