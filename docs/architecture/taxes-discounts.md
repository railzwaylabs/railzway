# Taxes & Discounts

Railzway handles global billing compliance and price adjustments through a flexible Tax and Discount system. This ensures that the final amount billed to a customer accounts for local regulations and promotional offers.

## Tax Rates

A `TaxRate` defines the percentage and behavior of a tax applied to an invoice.

### Inclusive vs. Exclusive Taxes

- **Exclusive Tax (Most Common)**: The tax is added *on top* of the subtotal.
    - Subtotal: $100
    - Tax (10%): $10
    - **Total**: $110
- **Inclusive Tax**: The tax is already *included* in the price.
    - Price: $100
    - Tax (10%): $9.09 (Calculated as `100 - (100 / 1.1)`)
    - **Total**: $100

### Global Applicability
Tax rates can be applied at different levels:
1. **Organization Default**: Applied to all invoices within an organization.
2. **Customer Level**: Specific taxes based on the customer's legal jurisdiction (e.g., VAT for EU customers).
3. **Plan level**: Specific taxes for certain types of services (e.g., digital services vs. hardware).

---

## Adjustments & Discounts

Railzway supports reducing or increasing the amount due through flexible Adjustment mechanisms.

### 1. Invoice Adjustments
Discounts are currently implemented as **Adjustment Lines** (`LineTypeAdjustment`) on an invoice. 
- **Flexibility**: Can be used for fixed-amount discounts, manual price corrections, or system-generated credits.
- **Order of Operations**: Adjustments are ideally applied to the subtotal **before** tax calculation, ensuring the customer is taxed only on the final payable amount.
- **Note**: While specialized "Coupon" entities are planned for future releases, the current engine relies on these dynamic adjustment lines for maximum flexibility.

### 2. Credit Balances (Ledger-backed)
Railzway supports a robust Credit system handled through the financial ledger.
- **Credit Grant**: Adding a balance to a customer (recorded as `SourceTypeCreditGrant`).
- **Credit Use**: Automatically applying available balance to an open invoice (recorded as `SourceTypeCreditUse`).
- **Auditability**: Since every credit movement is a ledger transaction, the balance is always traceable back to the original granting event (e.g., a refund or a promotional gift).

---

## The Footprint: Accuracy & Rounding

To maintain 100% financial accuracy:
- All financial values are stored in **Cents** (integers).
- Percentages are stored as `float64` to maintain high precision.
- Rounding occurs only at the very final step of the total calculation to prevent "penny drift" (where the sum of items doesn't match the total).

> [!CAUTION]
> Once a tax rate has been used on an issued invoice, it should be marked as `inactive` rather than edited or deleted. This preserves the financial history and audit trail for that historical period.
