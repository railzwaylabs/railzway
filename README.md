# Railzway

**Self-hosted billing engine for usage-based SaaS**

![Release CI](https://github.com/smallbiznis/railzway/actions/workflows/github-release.yml/badge.svg)
[![Docker Release](https://github.com/smallbiznis/railzway/actions/workflows/docker-release.yml/badge.svg)](https://github.com/smallbiznis/railzway/actions/workflows/docker-release.yml)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Release](https://img.shields.io/github/v/release/smallbiznis/railzway)
![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8?logo=go)

Railzway extracts billing concerns—usage metering, pricing, subscriptions, and invoicing—out of application code and into a **dedicated, self-hosted engine** with explicit boundaries.

> **Railzway determines what should be billed.**  
> **It does not execute payments.**

---

## Why Railzway?

Billing is a **financial truth system**, not a convenience feature.

In many systems, billing logic is:

- Scattered across application code
- Difficult to audit or reproduce
- Tightly coupled with payments and entitlements
- Fragile under scale and change

Railzway takes a deliberate approach:  
**Make billing boring, deterministic, and explainable.**

### Perfect For

- **AI/ML APIs** - Token-based pricing with multi-dimensional metering
- **Infrastructure Platforms** - Compute + storage + bandwidth billing
- **IoT/Hardware-as-a-Service** - Per-device, per-message pricing
- **Any SaaS** - Complex usage patterns beyond simple subscriptions

### Not For

- Simple subscription-only billing (use Stripe)
- Need merchant-of-record (use Paddle)
- Want fully-managed service (use Lago Cloud)

---

## What Railzway Is

Railzway is a **billing computation engine**, not an all-in-one billing platform.

### Core Capabilities

- **Subscription Management**Trialing, active, past_due, and canceled states with explicit lifecycle transitions
- **Usage Metering**Idempotent ingestion, deterministic aggregation, late and out-of-order handling
- **Smart Rate Limiting**
  Quota visibility, transparent limits, and informative error responses
- **Pricing Models**Flat-rate, tiered usage, and hybrid (base + usage) pricing with time-bound windows
- **Invoicing**Deterministic line-item generation, proration, and invoice state management
- **Multi-Tenancy**Organization-scoped isolation and authorization
- **Audit Trail**
  Immutable event log for all billing state changes
- **Payment Integrations**
  Built-in adapters for Stripe, Adyen, and Braintree, with an extensible provider interface
- **Taxation**
  Configurable tax behavior (inclusive/exclusive) and basic rate application
- **Entitlements**
  Billing-driven feature provisioning and sync capabilities

---

## Design Principles

Railzway is built around strict principles:

- **Deterministic by Design**Billing outputs are derived solely from persisted inputs and configuration.
- **Explicit Boundaries**Billing logic is separated from payments, identity, and infrastructure concerns.
- **Self-Hosted Ownership**Teams retain full control over billing data and logic.
- **Composable Primitives**
  Pricing behavior is modeled explicitly, not hidden in application code.

---

## Scope & Non-Goals

### What Railzway Is

Railzway is a **deterministic billing computation engine** designed for modern SaaS and platform systems.
It extracts billing logic from application code into a dedicated, auditable system with explicit boundaries.

### What Railzway Is Not

To preserve correctness and clarity, the following are **intentionally out of scope**:

#### Payment Execution

- No native credit card processing (delegates to adapters)
- **Stripe, Xendit, and Braintree Adapters** included for payment orchestration
- Extensible provider interface for additional gateways
- See [Payment Services Documentation](docs/payment-services.md)

#### Merchant of Record & Compliance

- No automated jurisdictional tax calculation (e.g. Avalara/Vertex)
- No PCI-DSS, PSD2, or regulatory automation
- Tax amounts stored as line items (basic rate calculation supported)
- *Future*: tax metadata structure

#### Dunning & Collections

- No retry logic for failed payments
- No automated email or recovery workflows
- Past-due state exists, recovery is external
- *Future*: webhook events

#### Revenue Recognition & Accounting

- No GAAP / IFRS compliance
- No deferred revenue tracking
- Cash-basis invoice generation only
- *Future*: ledger API integrations

#### Multi-Currency & FX

- Single currency per tenant
- No real-time FX handling
- *Future*: multi-currency via external FX providers

#### Entitlements & Feature Gating

- No access control or feature flags
- Application layer is responsible
- *Future*: entitlement sync via webhooks

#### Customer Self-Service UI

- No built-in customer portal
- API-first design
- *Future*: optional reference UI

#### Advanced Analytics & Reporting

- No MRR/ARR dashboards or cohort analysis
- Raw data available via API
- *Future*: Railzway Cloud analytics layer

#### Complex Proration Models

- Second-based precision proration
- Boundary rule: `[start inclusive, end exclusive]`
- *Future*: configurable proration strategies

---

## Why These Boundaries Matter

Billing errors cost money, trust, and legal risk.

By limiting scope, Railzway ensures:

- reproducible and auditable billing outputs,
- clear ownership of responsibilities,
- clean integration with best-of-breed tools,
- no black-box financial behavior.

Self-hosting is a **feature**, not a limitation:

- full data ownership,
- internal auditability,
- no vendor lock-in,
- compliance with data residency requirements.

---

## Architecture & Security

First-class documentation is provided:

- `ARCHITECTURE.md` — deterministic billing flows and trust boundaries
- `SECURITY.md` — security scope and assumptions
- `THREAT_MODEL.md` — lightweight threat model

---

## Deployment Model

Railzway is **self-hosted software**.

The adopting organization is responsible for:

- infrastructure and networking,
- TLS termination,
- secrets management,
- database operations and backups.

Railzway makes minimal assumptions about the runtime environment.

---

## Persistent Storage (Volumes)

Railzway uses `/var/lib/railzway` for persistent data storage. This directory should be mounted as a volume in production deployments.

### Volume Structure

```
/var/lib/railzway/
├── .instance_id              # Anonymous instance ID for telemetry
└── config/
    └── billing.yml           # Billing configuration (hot-reloadable)
```

### Docker Volume Mounting

**Docker Run:**

```bash
docker run -d \
  -v railzway-data:/var/lib/railzway \
  -p 8080:8080 \
  ghcr.io/smallbiznis/railzway:latest
```

**Docker Compose:**

```yaml
services:
  railzway:
    image: ghcr.io/smallbiznis/railzway:latest
    volumes:
      - railzway-data:/var/lib/railzway
      # Optional: Mount custom billing config
      - ./billing.yml:/var/lib/railzway/config/billing.yml:ro
    ports:
      - "8080:8080"

volumes:
  railzway-data:
```

**Kubernetes:**

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: railzway-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: railzway
spec:
  template:
    spec:
      containers:
      - name: railzway
        image: ghcr.io/smallbiznis/railzway:latest
        volumeMounts:
        - name: data
          mountPath: /var/lib/railzway
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: railzway-data
```

### Custom Billing Configuration

Create a `billing.yml` file to customize aging buckets and risk levels:

```yaml
billing:
  agingBuckets:
    - label: "0-30"
      minDays: 0
      maxDays: 30
    - label: "31-60"
      minDays: 31
      maxDays: 60
    - label: "60+"
      minDays: 61
      maxDays: null
  riskLevels:
    - level: "high"
      minOutstanding: 1000000  # $10,000 in cents
      minDays: 60
    - level: "medium"
      minOutstanding: 250000   # $2,500 in cents
      minDays: 31
    - level: "low"
      minOutstanding: 0
      minDays: 0
```

See `billing.yml.example` for a complete reference.

**Hot Reload:** Changes to `billing.yml` are automatically detected and applied without restart.

---

## Who Railzway Is For

Railzway is designed for teams that:

- are scaling beyond hardcoded billing logic,
- operate usage-based or hybrid pricing models,
- build long-lived SaaS or platform systems,
- value correctness, clarity, and auditability.

Railzway is **not a good fit** if you need:

- an all-in-one billing + payments + tax solution,
- no-code billing configuration,
- a managed service with instant SLA guarantees.

---

## Roadmap (High-Level)

**Current Features (v1.5.0)**

- Harden subscription and billing lifecycles
- Deterministic usage ingestion
- Invoice reproducibility guarantees
- End-to-end billing truth tests
- Stable APIs and semantic contracts

**Future Roadmap**

- Payment adapter interfaces
- Webhook event system
- Multi-currency support
- Tax metadata integration
- Optional **Railzway Cloud**

---

## Observability

Grafana dashboards expect the following metrics:

- `railzway_scheduler_job_runs_total`
- `railzway_scheduler_job_duration_seconds`
- `railzway_scheduler_job_timeouts_total`
- `railzway_scheduler_job_errors_total`
- `railzway_scheduler_batch_processed_total`
- `railzway_scheduler_batch_deferred_total`
- `railzway_scheduler_runloop_lag_seconds`

---

## License

Railzway is Open Source software under the **MIT License**.

You are free to use, modify, and distribute Railzway for any purpose, including commercial use, without restrictions.

## Telemetry

Railzway collects **anonymous usage metrics** to help us understand adoption and prioritize improvements.

### What We Collect

- **Instance ID**: Random UUID (not tied to your identity)
- **Version Info**: Railzway version and OS
- **Aggregate Counts**: Total organizations, subscriptions, invoices
- **Feature Flags**: Which features are enabled (e.g., tiered pricing)

### What We DON'T Collect

- ❌ Personal information (emails, names, IP addresses)
- ❌ Financial data (invoice amounts, customer data)
- ❌ Usage event contents
- ❌ Any identifiable business information

### Why Telemetry?

This helps us:
- Understand how many instances are running (project health)
- Prioritize features based on actual usage
- Identify common error patterns for stability improvements

### Opt-Out

Telemetry is **enabled by default**. To disable:

```bash
TELEMETRY_ENABLED=false
```

Or set in your `.env` file:

```env
TELEMETRY_ENABLED=false
```

### Transparency

- View telemetry code: [`internal/cloudmetrics/`](internal/cloudmetrics/)
- All data is anonymized and aggregated
- We respect your privacy and data ownership

---

## Documentation

📚 Documentation source: `./docs/index.md`

> **Railzway aims to make billing boring, predictable, and explainable**
> so teams can focus on building their products.
