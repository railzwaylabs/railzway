# Admin Cypress E2E Standard

Default Cypress e2e specs are billing-critical UI tests. They are not intended to cover every admin page. A resource belongs in this suite only when a broken UI flow can directly affect billing correctness, invoice totals, usage rating, ledger integrity, tax calculation, subscription lifecycle, or test-clock simulation.

Tests should drive the UI for the behavior under test. API calls are allowed only for authentication, organization selection, and deterministic fixture setup.

`cypress.config.ts` intentionally scopes `specPattern` to the billing folders below. Adding a new folder under `cypress/e2e` will not make it part of the default e2e suite unless it is a billing-critical resource and the config is updated.

## Structure

- `catalog/`: products, plans, prices, meters, and features.
- `customers/`: customer create/edit/list flows.
- `subscriptions/`: subscription create/edit/list flows.
- `invoices/`: invoice generation, totals, and lifecycle management.
- `usage/`: usage list filters, CSV upload, and idempotency UI behavior.
- `ledger/`: manual journal entry and ledger list behavior.
- `taxes/`: tax rate list/create flows.
- `rating/`: rating results and aggregate views.
- `test-clocks/`: test clock management UI.
- `workflows/`: multi-resource business flows that intentionally span pages.

## Scenario Rules

- Every included billing resource should have at least one positive path and one negative or guardrail path.
- Positive paths must submit through UI controls, then assert visible UI state.
- Negative paths should assert disabled submits, validation messages, toasts, or failed UI-triggered requests.
- Avoid API-only specs under Cypress e2e. Move API contract coverage to backend/API tests instead.
- Fixture APIs should stay inside setup blocks and should not be the primary behavior being tested.
- Do not add auth, dashboard, settings, or generic admin shell checks here unless they are part of a billing-critical workflow.

## Baseline Coverage

- Catalog: product, plan, meter, and feature create validation plus happy paths.
- Customers: create, filter, edit, required field validation, and email validation.
- Subscriptions: seeded fixture selection through UI and required-field validation.
- Invoices: generation guardrails, UI total generation, open/pay lifecycle.
- Ledger: balanced transaction success, empty entry validation, unbalanced entry rejection.
- Usage: CSV upload success, missing file guard, filters, duplicate idempotency UI behavior.
- Taxes: list navigation, create success, required field validation.
- Rating: rating results and aggregate navigation with filters.
- Test clocks: create/pause/resume/audit trail and invalid frozen-time guard.
- Workflows: audit visibility, org isolation, and catalog-to-subscription integrity.

## Out Of Scope For Default E2E

- Login page edge cases.
- Dashboard smoke checks.
- API key management and general settings.
- Generic autocomplete/search smoke checks that do not assert billing behavior.
- Read-only payments list filtering when invoice lifecycle tests already cover payment creation.

Those can be covered by unit/component tests, backend/API tests, or separate smoke suites. They should not expand the default admin Cypress e2e surface.
