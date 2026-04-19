# Metering & Aggregation Logic

The metering system is the engine that transforms raw interactions (Usage Events) into quantities that can be billed. This document explains how Railzway handles event ingestion and aggregation.

## Core Concepts

### 1. Usage Event
A raw record of an activity performed by a customer.
- **CustomerID**: Who performed the action.
- **MeterCode**: Which meter tracks this action.
- **Value**: The numeric value of the event (e.g., `1.5` hours, `500` MB).
- **RecordedAt**: When the action occurred (used for billing period matching).
- **IdempotencyKey**: A unique identifier provided by the client to prevent double-billing if the same event is sent multiple times.

### 2. Meter
A definition of *how* usage should be measured and aggregated over time.

---

## Aggregation Methods

Railzway supports multiple aggregation strategies to accommodate different business models:

| Method | Description | Example Use Case |
| :--- | :--- | :--- |
| `SUM` | Adds up all event values within the billing period. | Total GB stored, Total SMS sent. |
| `COUNT` | Counts the number of events, regardless of value. | Total API calls, Total logins. |
| `MAX` | Takes the highest value recorded in the period. | Peak concurrent users, Max storage peak. |
| `UNIQUE_COUNT` | Counts the number of unique occurrences of a specific property. | Unique monthly active users (MAU). |
| `LATEST` | Uses the value from the most recent event. | Current number of seats, Current plan level. |

---

## The Ingestion Pipeline

1. **Validation**: Railzway checks if the `OrgID`, `CustomerID`, and `MeterID` exist and are active.
2. **Idempotency Check**: The system checks if an event with the same `IdempotencyKey` has already been processed for the organization.
3. **Persistence**: The event is saved to the `usage_events` table with a `pending` status.
4. **Aggregation (Periodic/On-demand)**: During the "Rating" phase, the system queries events within the specific `PeriodStart` and `PeriodEnd`, applying the Meter's `Aggregation` logic to produce a single `UsageAggregate`.

## High-Volume Ingestion
Railzway is designed to handle high-volume event streams. For massive scale, events can be buffered or batched before being sent to the `/ingest` endpoint.

> [!IMPORTANT]
> Always provide an `IdempotencyKey`. In a distributed system, network retries are common; without this key, you risk overcharging your customers.
