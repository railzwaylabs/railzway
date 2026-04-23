import { generateInvoice, seedBillingFixture, switchOrg, visitOrgPath } from "../../support/billing";

describe("Admin UI Invoice Lifecycle", () => {
  beforeEach(() => {
    cy.login();
  });

  it("opens and pays invoice from manage screen", () => {
    const uid = Date.now();
    let orgId = "";
    let invoiceId = "";
    let invoiceNumber = "";

    cy.createOrg(`UI Lifecycle Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    });

    cy.then(() => {
      return seedBillingFixture({
        uid,
        prefix: `ui_lifecycle_${uid}`,
        flatAmountCents: 3000,
        taxPercentage: 10
      }).then((fixture) => {
        return generateInvoice(fixture.subscriptionId, fixture.periodStart, fixture.periodEnd);
      }).then((invoice) => {
        invoiceId = invoice.id;
        invoiceNumber = invoice.number;
      });
    });

    cy.then(() => {
      visitOrgPath(orgId, `/invoices/${invoiceId}/manage`);
      cy.contains(".page-content", invoiceNumber).should("be.visible");
      cy.intercept("POST", `/admin/v1/invoices/${invoiceId}/open`).as("openInvoice");
      cy.get("[data-testid=\"invoices-manage-open\"]").click();
      cy.wait("@openInvoice").its("response.body.invoice.status").should("eq", "open");

      cy.intercept("POST", `/admin/v1/invoices/${invoiceId}/pay`).as("payInvoice");
      cy.get("[data-testid=\"invoices-manage-pay\"]").click();
      cy.wait("@payInvoice").its("response.body.invoice.status").should("eq", "paid");
    });
  });
});
