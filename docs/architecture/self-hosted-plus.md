# High Availability & Failure Model

**Product:** Railzway Self-Hosted Plus
**Audience:** Enterprise Engineering, SRE, Security, Procurement
**Scope:** Application availability, failure handling, and recovery model

---

## 1. Purpose

This document explains how **Railzway Self-Hosted Plus** achieves High Availability (HA), how failures are handled, and what guarantees (and non-guarantees) are provided.

The goal is to provide **clear operational expectations**, not marketing claims.

---

## 2. Architectural Overview

Railzway is implemented as a **stateless modular monolith** at the application layer.

* Multiple identical Railzway instances can run concurrently
* All persistent state is stored in an external database
* No instance holds exclusive in-memory state

High Availability is achieved through **horizontal replication**, not service decomposition.

---

## 3. Deployment Model

Typical HA deployment:

* ≥ 2 Railzway instances
* Load balancer in front of instances
* Shared external database (PostgreSQL or equivalent)

Instances are **active-active** and can all serve traffic simultaneously.

---

## 4. Stateless Application Layer

Railzway application instances are stateless:

* No session affinity required
* No local persistence required for correctness
* Instance termination does not cause data loss

This allows:

* Horizontal scaling
* Rolling upgrades
* Fast replacement of failed nodes

---

## 5. Failure Scenarios & Behavior

### 5.1 Application Instance Failure

**Scenario:** One Railzway instance crashes or becomes unreachable.

**Behavior:**

* Load balancer routes traffic to remaining healthy instances
* No data loss occurs
* Ongoing requests may retry or fail fast depending on client configuration

**Impact:**

* Reduced capacity only
* No functional degradation

---

### 5.2 Load Balancer Failure

**Scenario:** Load balancer failure.

**Behavior:**

* Mitigated through customer-chosen HA load balancer (cloud or on-prem)
* Railzway does not embed a proprietary load balancer

**Impact:**

* Depends on customer infrastructure design

---

### 5.3 Database Failure

**Scenario:** Primary database becomes unavailable.

**Behavior:**

* Railzway returns errors for operations requiring persistence
* No silent data corruption occurs

**Expectation:**

* Database HA (replication / managed service) is required for production deployments

---

### 5.4 Network Partition

**Scenario:** Partial network connectivity between instances and database.

**Behavior:**

* Requests fail explicitly
* No split-brain behavior
* No local write buffering

This ensures **consistency over availability** for billing correctness.

---

## 6. Background Jobs & Schedulers

Railzway may execute background jobs such as:

* Billing cycle transitions
* Scheduled rating or invoice generation

### Coordination Model

* Background jobs use **database-backed coordination** (e.g. advisory locks)
* At most one instance executes a given job at a time
* Other instances remain idle for that job

### Failure Handling

* If the executing instance crashes, locks are released
* Another instance may safely resume execution

This prevents:

* Duplicate billing
* Partial execution
* Race conditions

---

## 7. Upgrade & Maintenance Strategy

### Rolling Upgrades

* Instances are upgraded one at a time
* Remaining instances continue serving traffic
* No global downtime required

### Backward Compatibility

* Schema migrations are designed to be backward compatible where possible
* Destructive migrations are explicitly documented

---

## 8. License System & Availability

License validation:

* Is performed locally
* Does not require continuous network access
* Does not affect core billing functionality

License expiration or validation failure:

* Disables premium features only
* Does not cause service shutdown

---

## 9. Non-Goals & Explicit Exclusions

Railzway does **not** provide:

* Cross-region active-active replication
* Built-in database replication
* Guaranteed RPO/RTO values

These concerns are handled at the infrastructure layer chosen by the customer.

---

## 10. Operational Responsibilities

### Railzway Provides:

* Deterministic billing behavior
* Stateless application design
* Safe concurrency mechanisms

### Customer Provides:

* Load balancer
* Database HA
* Backup & restore strategy
* Monitoring and alerting

---

## 11. Summary

Railzway Self-Hosted Plus achieves High Availability through:

* Stateless modular monolith design
* Horizontal replication
* Externalized state management
* Explicit failure handling

This approach minimizes operational complexity while providing predictable and auditable behavior suitable for enterprise environments.

---

*This document intentionally prioritizes clarity and correctness over marketing claims.*
