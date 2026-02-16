# ADR-003: API Versioning Strategy

[← Back to Documentation Index](../index.md)

**Status:** Accepted

**Date:** 2026-02-16

**Scope:** Public API Surface

---

## Context

As Railzway evolves, we need a mechanism to introduce breaking changes to the API (e.g., renaming fields, changing resource structures) without disrupting existing consumers.

Without a clear versioning strategy:
- Every change must be backward compatible forever.
- The API becomes cluttered with deprecated fields.
- Clients break unexpectedly if validation rules change.

We considered several approaches:
1.  **URL Path Versioning** (`/api/v1/resource`)
2.  **Header Versioning** (`Accept-Version: v1`)
3.  **Media Type Versioning** (`Accept: application/vnd.railzway.v1+json`)

---

## Decision

We will use **URL Path Versioning** for major API versions.

- Current Version: `/api/v1/*`
- Future Major Versions: `/api/v2/*`, `/api/v3/*`

Minor changes (additive non-breaking changes) will appear within the existing version path without changing the version number.

---

## Rationale

### 1. Distinct & Explicit
The version is explicitly visible in the URL. Developers can see at a glance which version they are using. It is impossible to "forget" to send a version header and accidentally get the wrong version.

### 2. Cache Friendly
CDNs and proxies can easily cache responses based on the URL path. Header-based versioning often requires complex `Vary` configurations that are prone to errors.

### 3. Developer Experience
It is the standard for modern public APIs (e.g., Stripe, Twilio, Google Cloud). It requires zero special tooling in HTTP clients; standard `curl` or browser calls work immediately.

### 4. Implementation Simplicity
Routing logic in Go (specifically `gin`) handles path prefixes natively. Grouping handlers by version (`v1 := r.Group("/v1")`) keeps the code organized and isolated.

---

## Consequences

### Positive
- **Stability**: Existing clients on `/v1` are insulated from breaking changes in `/v2`.
- **Clarity**: Documentation and code are clearly separated by major version.
- **Parallelism**: Multiple versions can run simultaneously in the same application instance.

### Negative
- **URL Changes**: Clients must update their base URLs to migrate to a new version.
- **Code Duplication**: Major upgrades may require duplicating handlers/models to maintain independence, though shared logic can be extracted to internal service layers.

---

## Deprecation Policy

When a new major version is released:
1.  The previous version enters **Deprecated** status.
2.  It remains supported for a defined grace period (e.g., 12 months) to allow migration.
3.  Security fixes are backported, but new features are primarily added to the new version.
