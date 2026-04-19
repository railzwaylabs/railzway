# Ledger & Double-Entry Accounting

Railzway is built on a double-entry ledger foundation. Every financial event—from a usage-based charge to a payment settlement—is recorded as an immutable set of ledger entries. This ensures data integrity, auditability, and financial accuracy.

## Why a Ledger?
Traditional billing systems often store "running balances" which can be prone to race conditions or silent data corruption. Railzway records every movement of value as a distinct transaction. The balance is a derivation of all ledger entries.

## Core Entities

### 1. Ledger Account
A categorization for financial entries. Railzway follows standard accounting categories:
- **Assets**: Money owed to the company (e.g., Accounts Receivable).
- **Liability**: Money the company owes (e.g., Prepaid Credits, Tax Payable).
- **Income**: Revenue earned (e.g., Sales Revenue).
- **Expense**: Costs incurred (e.g., Payment Processing Fees).

### 2. Ledger Transaction
A header for a financial event. It links to a source (e.g., an Invoice, a Payment) and contains the metadata for that event.

### 3. Ledger Entry
A single line in a transaction. In **Double-Entry** accounting:
- A single transaction must have at least two entries.
- The sum of **Debits** must equal the sum of **Credits** (for balanced transactions).
- **Debit**: Increases Assets/Expenses, Decreases Liability/Income.
- **Credit**: Increases Liability/Income, Decreases Assets/Expenses.

---

## Example Flow: Invoice Generation

When an invoice for $100 is "Opened", Railzway records the following transaction:

| Account | Entry Type | Amount | Explanation |
| :--- | :--- | :--- | :--- |
| `accounts_receivable` | `DEBIT` | $100 | Increases what the customer owes us. |
| `sales_revenue` | `CREDIT` | $100 | Increases the income we've earned. |

## Example Flow: Payment Received

When the customer pays the $100 invoice:

| Account | Entry Type | Amount | Explanation |
| :--- | :--- | :--- | :--- |
| `cash_at_bank` | `DEBIT` | $100 | Increases our actual cash asset. |
| `accounts_receivable` | `CREDIT` | $100 | Decreases the debt the customer owed. |

---

## Auditability & Reliability

- **Immutable**: Ledger entries cannot be edited or deleted once posted. Adjustments are made via new "Reversal" or "Adjustment" transactions.
- **Traceability**: Every transaction contains a `SourceType` and `SourceID` (e.g., `invoice_id`), allowing you to trace a ledger line back to the specific usage events or subscription rules that triggered it.
- **Idempotency**: Every financial operation uses an `IdempotencyKey` to ensure that a ledger entry is never duplicated due to network or logic retries.
