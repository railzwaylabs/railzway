# Billing Reconciliation

Reconciliation is the proactive defense mechanism of the Railzway engine. It ensures that usage data, rating results, and financial ledger entries remain synchronized and accurate over time.

## The Reconciliation Vision: "Trust but Verify"

While the billing pipeline is designed to be ACID-compliant and idempotent, distributed systems can encounter edge cases (e.g., race conditions, manual database interventions, or logic drift). The reconciliation engine acts as a secondary verification layer.

---

## Current Implementation: Invoice Reconciliation

Railzway runs a periodic background scheduler (`ReconciliationScheduler`) that performs a triple-match check for every issued invoice within a specific window (default: last 7 days).

### 1. Usage Match: Rating vs. Invoice
- **Calculation**: Sum of all `amount_cents` in `rating_results` for the invoice period.
- **Verification**: Must match the sum of `amount_cents` for all `usage` type line items in the `invoice_items` table.
- **Goal**: Detect if a rating result was omitted from an invoice or if usage was double-counted.

### 2. Financial Match: Ledger vs. Invoice
- **Calculation**: Sum of all `credit` entries in the `ledger_entries` table associated with the billing cycle transaction for that invoice.
- **Verification**: Must match the `total_cents` of the `invoice`.
- **Goal**: Detect if the financial record in the ledger has drifted from the customer's legal invoice.

---

## Proposed Upgrades (Roadmap)

To reach enterprise-grade reliability, the following reconciliation checks are planned:

### A. Tax Verification
Re-calculate the expected tax for every invoice based on the customer's jurisdiction and the subtotal. Any discrepancy between the calculated tax and the `tax_cents` field will trigger an audit alert.

### B. Payment Settlement Check
Verify that the `amount_paid_cents` on an invoice matches the total sum of `SourceTypePayment` transactions recorded in the ledger for that specific invoice.

### C. Subscription Integrity
Cross-reference `subscription` type line items with the historical `plan_price` snapshots to ensure recurring fees were generated with the correct base price.

---

## Audit & Alerting

When a mismatch is detected:
1. **Logging**: A `WARN` level log is emitted with full context (Invoice ID, Expected vs. Actual values).
2. **Audit Log**: A `reconciliation.mismatch` entry is created in the `audit_logs` table.
3. **Admin Visibility**: Discrepancies are flagged in the Admin UI for manual review and correction.

> [!IMPORTANT]
> Reconciliation is a **Read-Only** operation. It identifies issues but does not automatically change financial records. Correction must be done via manual adjustment or reversal transactions to maintain the audit trail.
