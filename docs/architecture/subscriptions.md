# Subscriptions & Proration

Managing the lifecycle of a subscription is one of the most complex tasks in billing. Railzway automates the calculation of active windows and proration factors to ensure precise billing even when plans change mid-cycle.

## Subscription Lifecycle

A subscription defines a customer's relationship with a product over time.
- **Trialing**: The customer is using the service for free before starting a paid cycle.
- **Active**: The customer is currently subscribed and generating invoices.
- **Past Due**: A payment failed, and the system is attempting retries (dunning).
- **Canceled**: The subscription has ended, and no further billing will occur.

---

## Proration: Precision in Mid-Cycle Changes

Proration occurs when a customer's plan changes (upgrades, downgrades, or cancellations) before the billing period ends. Railzway calculates the "Effective Window" for every item to determine the exact amount to bill.

### 1. Effective Window Calculation
Railzway intersects three time ranges to find the billable window:
- **Billing Period**: The default 30-day or 1-year window.
- **Subscription Lifetime**: When the sub started and when it was canceled.
- **Price/Item Lifetime**: When a specific price was added or removed from the plan.

### 2. The Proration Factor
Once the active window is known, Railzway calculates a factor:
`Factor = (End - Start) in seconds / (PeriodEnd - PeriodStart) in seconds`

**Example**:
- Billing Period: Oct 1 to Oct 31 (30 days).
- Customer upgrades on Oct 16.
- **Old Plan Factor**: 15 days / 30 days = `0.5`.
- **New Plan Factor**: 15 days / 30 days = `0.5`.

The system will then credit back 50% of the old plan and charge 50% of the new plan.

---

## Multi-Price Subscriptions
A single Railzway subscription can contain multiple "Prices". This allows for complex hybrid models:
- **Subscription Price**: $100/mo (Prorated if changed).
- **Add-on Price**: $20/mo (Prorated if added mid-month).
- **Usage Price**: $0.10/unit (Never prorated, as it's based on actual event volume).

## Handling Cancellations
When a subscription is canceled:
- **Cancel at Period End**: The subscription remains `active` until the end of the current cycle, then moves to `canceled`. No proration is needed.
- **Cancel Immediately**: The subscription is terminated instantly. Railzway calculates the proration factor for the unused time and generates a credit for the customer.

> [!IMPORTANT]
> Usage-based prices are **not** prorated. They are always rated based on the actual quantity of events recorded between the `start` and `end` of the active window.
