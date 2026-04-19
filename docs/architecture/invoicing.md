# Invoicing & Rating Engine

The Invoicing module is responsible for aggregating billing data and generating financial documents for customers. It works closely with the Rating module to calculate the cost of usage.

## The Rating Process (Usage -> Money)

Before an invoice can be generated, Railzway must "Rate" the usage. The process follows these steps:
1. **Aggregation**: Collect all `UsageEvents` for a customer and meter within a specific window (e.g., one month).
2. **Pricing Look-up**: Find the `PlanPrice` associated with that meter in the customer's active subscription.
3. **Calculation**: Compute the final amount based on the price type:
    - **Flat/Fixed**: Apply the base amount.
    - **Per-unit**: Multiply aggregated quantity by unit price.
    - **Tiered**: Apply volume or graduated tiers.
4. **Rating Result**: The calculated amount and quantity are saved as a `RatingResult`.

---

## Invoice Lifecycle

A Railzway invoice moves through several statuses:

| Status | Description | Financial Impact |
| :--- | :--- | :--- |
| `DRAFT` | The default state during a billing cycle. Quantities and amounts can change as more usage is ingested. | None (Informational only). |
| `OPEN` | The invoice is finalized and issued to the customer. It becomes a legal financial document. | Records `Debit` to Accounts Receivable in the Ledger. |
| `PAID` | The payment has been successfully recorded and settled. | Records `Credit` to Accounts Receivable and `Debit` to Cash. |
| `VOID` | The invoice was issued in error and is now cancelled. | Records a reversal of the `OPEN` ledger transaction. |
| `UNCOLLECTIBLE` | The customer failed to pay, and the debt is "written off". | Records an expense for bad debt. |

---

## Invoice Structure

### Header
Contains overall totals (`Subtotal`, `Tax`, `Total`), currency, billing period, and the unique `Invoice Number`.

### Line Items
Each invoice is composed of multiple `InvoiceItem` entries:
- **Subscription**: Fixed recurring fees based on the plan.
- **Usage**: Variable fees calculated by the Rating engine.
- **Adjustment**: Manual or system-generated corrections.
- **Credit**: Applied balances or discounts.
- **Tax**: Government levies calculated on the subtotal.

---

## Checksums and Integrity
To ensure financial records are not tampered with, Railzway generates a `Checksum` for every finalized invoice. Any modification to the line items or totals would result in a checksum mismatch, flagging the record for audit.

> [!TIP]
> Use the **Draft** status to provide real-time cost estimates to your customers within your application dashboard before the billing cycle ends.
