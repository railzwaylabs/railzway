# Invoice Service (`apps/invoice`)

The **Invoice Service** is the **Customer Plane** of Railzway. It is responsible for public-facing interactions related to billing documents.

## Purpose

- **Public Invoice Rendering**: Hosting the persistent "View Invoice" page sent to customers via email.
- **Checkout / Payment Collection**: Secure UI for entering credit card details to pay an invoice.
- **PDF Generation**: (Optional) Rendering invoices to PDF for download.

## Security

This service is conceptually separate from the Admin Control Plane to isolate risk. It is exposed to the public internet but has restricted access (e.g., via signed tokens or public UUIDs) compared to the Admin API.

## Architecture

Similar to Admin, this bundles a lightweight React UI with a Go backend.

## Running

```bash
# Docker
docker run -p 8080:8080 ghcr.io/<org>/railzway-invoice:latest
```
