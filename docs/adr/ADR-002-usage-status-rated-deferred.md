
## **ADR-002: Why `usage_events.status.rated` Is Deferred**

**Status:** Accepted

**Date:** 2026-01-02

**Scope:** Usage lifecycle & billing pipeline

---

### Context

The usage lifecycle defines the following statuses:

* `accepted`
* `enriched`
* `rated`
* `invalid`
* `unmatched_meter`
* `unmatched_subscription`

Currently:

* Usage ingestion sets `status = accepted`.
* The snapshot worker transitions usage events to `enriched` or `unmatched_*` after resolving meter and subscription context.
* No component updates `usage_events.status` to `rated`.
* The rating pipeline produces `rating_results`, which represent the billable output of usage aggregation.

As a result, `UsageStatusRated` exists in the domain model but is not materialized in the database.

---

### Decision

The `UsageStatusRated` state is intentionally deferred and not produced at this stage.

Billing correctness is determined by the successful creation of `rating_results`, not by a per-event status flag on `usage_events`.

Until a dedicated post-rating stage exists, `enriched` usage events are treated as **structurally billable input** and are used as input to the rating pipeline.

This does **not** imply that `enriched` represents final billing state.

---

### Rationale

* Rating is a window-based, transactional process that may succeed or fail independently of individual usage events.
* Marking usage events as `rated` before rating results are committed risks inconsistent billing state.
* Deferring `rated` avoids coupling snapshot logic with pricing and billing concerns.
* This design aligns with Stripe-like billing semantics, where usage recording is decoupled from billing finalization and billing truth is derived from downstream artifacts.

---

### Consequences

* Aggregation queries currently filter on `status = enriched`.
* `usage_events.status` represents  **processing and validation state** , not final billing state.
* `rating_results` is the  **financial source of truth** .
* Introducing `rated` in the future will not break existing data or queries.

---

### Non-Goals

This ADR does not define:

* the implementation of a rating engine,
* pricing calculation rules,
* or when `UsageStatusRated` will be introduced.

---

### Future Considerations

A post-rating step may be introduced to:

* mark usage events as `rated` after rating results are successfully committed, or
* derive `rated` state implicitly from `rating_results` without mutating `usage_events`.

No change is required to snapshot logic.
