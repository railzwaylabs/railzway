import { generateInvoice, seedBillingFixture, switchOrg } from "../../support/billing";

describe("Admin API Financial Audit Trail", () => {
  beforeEach(() => {
    cy.login();
  });

  it("records audit logs for critical financial endpoint actions", () => {
    const uid = Date.now();
    let orgId = "";

    cy.createOrg(`API Audit Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    });

    cy.then(() => {
      return seedBillingFixture({
        uid,
        prefix: `api_audit_${uid}`,
        flatAmountCents: 2100
      }).then((fixture) => {
        return generateInvoice(fixture.subscriptionId, fixture.periodStart, fixture.periodEnd).then((invoice) => {
          const invoiceId = invoice.id;
          return cy.csrfRequest("POST", `/admin/v1/invoices/${invoiceId}/open`).then(() => {
            return cy.csrfRequest("POST", `/admin/v1/invoices/${invoiceId}/pay`, {
              reason: "API audit payment"
            });
          }).then(() => {
            return cy.csrfRequest("POST", "/admin/v1/ledger/transactions", {
              currency: "USD",
              source_type: "manual_adjustment",
              source_id: `api_audit_manual_${uid}`,
              entries: [
                { account_code: "1000_cash", entry_type: "debit", amount_cents: 500, currency: "USD" },
                { account_code: "4000_revenue", entry_type: "credit", amount_cents: 500, currency: "USD" }
              ]
            });
          }).then(() => {
            return cy.csrfRequest("POST", "/admin/v1/test-clock", {
              current_time: "2026-03-20T00:00:00Z",
              status: "active"
            });
          });
        });
      });
    });

    [
      "invoice.generate",
      "invoice.open",
      "invoice.pay",
      "ledger.transaction.create",
      "testclock.upsert"
    ].forEach((action) => {
      cy.csrfRequest("GET", `/admin/v1/audit-logs?page_size=50&action=${encodeURIComponent(action)}`).then((resp) => {
        const actions = (resp.body.logs ?? []).map((log: { action: string }) => log.action);
        expect(actions).to.include(action);
      });
    });
  });
});
