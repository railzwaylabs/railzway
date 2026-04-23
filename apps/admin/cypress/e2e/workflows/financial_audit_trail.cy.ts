import { generateInvoice, seedBillingFixture, switchOrg, visitOrgPath } from "../../support/billing";

describe("Admin UI Financial Audit Trail", () => {
  beforeEach(() => {
    cy.login();
  });

  it("shows audit log entries in the UI for critical financial operations", () => {
    const uid = Date.now();
    let orgId = "";
    let invoiceId = "";

    cy.createOrg(`UI Audit Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    });

    cy.then(() => {
      return seedBillingFixture({
        uid,
        prefix: `ui_audit_${uid}`,
        flatAmountCents: 2100
      }).then((fixture) => {
        return generateInvoice(fixture.subscriptionId, fixture.periodStart, fixture.periodEnd).then((invoice) => {
          invoiceId = invoice.id;
          return cy.csrfRequest("POST", `/admin/v1/invoices/${invoiceId}/open`);
        }).then(() => {
          return cy.csrfRequest("POST", `/admin/v1/invoices/${invoiceId}/pay`, {
            reason: "UI audit payment"
          });
        }).then(() => {
          return cy.csrfRequest("POST", "/admin/v1/test-clocks", {
            frozen_time: Math.floor(Date.parse("2026-03-20T00:00:00Z") / 1000),
            name: `Audit Clock ${uid}`,
            status: "active",
          });
        });
      });
    });

    cy.then(() => {
      cy.intercept("GET", "/admin/v1/audit-logs*").as("auditLogs");
      visitOrgPath(orgId, "/audit-logs");
      cy.wait("@auditLogs");
    });

    ["invoice.generate", "invoice.open", "invoice.pay", "testclock.create"].forEach((action) => {
      cy.get("[data-testid=\"audit-logs-filter-action\"]").clear().type(action);
      cy.get("[data-testid=\"audit-logs-apply\"]").click();
      cy.wait("@auditLogs");
      cy.contains(".data-table", action).should("be.visible");
    });
  });
});
