# Payment Services & Orchestration

Railzway has expanded its scope to include **Payment Orchestration**, allowing it to not only calculate bills but also facilitate the collection of payments through external providers.

> **Principle**: Railzway orchestrates payments; it does not process them. It relies on specialized Payment Service Providers (PSPs) like Xendit and Stripe for the actual fund movement.

---

## Architecture Overview

The payment service layer handles:
1.  **Payment Method Tokenization**: Storing references to payment methods (cards, e-wallets) securely stored by PSPs.
2.  **Smart Routing**: Deciding which PSP to use based on customer location and currency.
3.  **Payment Collection**: Triggering charges for finalized invoices (future scope: subscription collections).
4.  **Reconciliation**: Processing webhooks to update invoice and ledger states.

### Dual Provider Strategy

Railzway adopts a multi-provider strategy to optimize for coverage and fees:

| Provider | Target Market | Currencies | Features |
|----------|---------------|------------|----------|
| **Xendit** | Southeast Asia (primary) | IDR, PHP, THB, MYR, VND | Local cards, Virtual Accounts, E-Wallets (OVO, Dana, GCash) |
| **Stripe** | Global / Rest of World | USD, EUR, SGD, etc. | Global cards, extensive international support |

---

## Payment Method Configuration

Railzway uses a **Configuration-Driven Routing** approach. Instead of hardcoding "Use Xendit for Indonesia", admins define rules in the system.

### Configuration Model
A `PaymentMethodConfig` defines:
- **Display Name**: e.g., "BCA Virtual Account"
- **Provider**: `xendit` or `stripe`
- **Availability Rules**: JSON rules defining where this method is shown.
  - `countries`: List of ISO 2-letter codes (e.g., `["ID", "PH"]`) or `["*"]`.
  - `currencies`: List of ISO 3-letter codes (e.g., `["IDR"]`).
- **Priority**: Determines sort order in the UI.

### Example Rules
1.  **Global Card (Stripe)**:
    - Countries: `["*"]`
    - Currencies: `["USD", "EUR", "SGD"]`
    - Priority: 100
2.  **Indonesia Card (Xendit)**:
    - Countries: `["ID"]`
    - Currencies: `["IDR"]`
    - Priority: 95 (Shown after global? Or before? Usually local first if specific. Adjust priority as needed).
3.  **GoPay (Xendit)**:
    - Countries: `["ID"]`
    - Currencies: `["IDR"]`
    - Priority: 90

The API `GET /payment-methods/available?country=ID&currency=IDR` filters these configs to return only relevant methods to the frontend.

---

## Integration Guide

### 1. Xendit Setup
Used for SEA markets.

**Prerequisites**:
- Xendit Account (Test or Live).
- API Key (Secret Key) with write permission.
- Webhook verification token (Callback Token).

**Configuration**:
In **Admin UI > Settings > Payment Providers > Xendit**:
- **API Key**: Enter your Xendit Secret Key.
- **Webhook Secret**: Enter your Xendit Callback Token.

**Webhooks**:
Configure your Xendit dashboard to send webhooks to:
`POST https://your-railzway-instance.com/webhooks/xendit`

### 2. Stripe Setup
Used for global markets.

**Prerequisites**:
- Stripe Account.
- Secret Key (`sk_...`).
- Webhook Signing Secret (`whsec_...`).

**Configuration**:
In **Admin UI > Settings > Payment Providers > Stripe**:
- **API Key**: Enter your Stripe Secret Key.
- **Webhook Secret**: Enter your Stripe Webhook Signing Secret.

**Webhooks**:
Configure your Stripe dashboard to send events to:
`POST https://your-railzway-instance.com/webhooks/stripe`
Events to subscribe to:
- `payment_intent.succeeded`
- `payment_intent.payment_failed`
- `charge.refunded`

---

## API Reference

### Customer Endpoints
- **GET /payment-methods/available**: List configured payment methods matching query params (`country`, `currency`).
- **GET /customers/:id/payment-methods**: List saved payment methods for a customer.
- **POST /customers/:id/payment-methods**: Attach a payment method (requires provider token).
- **POST /customers/:id/payment-methods/:pm_id/default**: Set default payment method.
- **DELETE /customers/:id/payment-methods/:pm_id**: Detach/remove payment method.

### Admin Endpoints
- **GET /admin/payment-method-configs**: List routing rules.
- **POST /admin/payment-method-configs**: Create a new rule.
- **PUT /admin/payment-method-configs/:id**: Update a rule.
- **POST /admin/payment-method-configs/:id/toggle**: Enable/Disable a rule.

---

## Troubleshooting

### "Payment Method Not Found"
- Check if the customer has a default payment method set.
- Check if the payment method config for that provider is active.

### "No Available Payment Methods"
- Check the `country` and `currency` params passed to the API.
- Verify `PaymentMethodConfigs` exist for that combination in Admin UI.
- Ensure the configs are set to `Active`.

### Webhook Verification Failures
- Verify that the **Webhook Secret** in Payment Providers config matches exactly what is in the Provider's dashboard.
- For Xendit, ensure you are using the "Callback Token", not the API Key.
