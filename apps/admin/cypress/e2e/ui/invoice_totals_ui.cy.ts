import { seedBillingFixture, selectAutocomplete, switchOrg, visitOrgPath } from "../../support/billing";

describe("Admin UI Invoice Totals", () => {
  beforeEach(() => {
    cy.login();
  });

  it("generates invoice from UI and lands on manage page with expected total", () => {
    const uid = Date.now();
    const expectedTotal = 2750;
    let orgId = "";
    let subscriptionId = "";

    cy.createOrg(`UI Invoice Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    });

    cy.then(() => {
      return seedBillingFixture({
        uid,
        prefix: `ui_invoice_${uid}`,
        flatAmountCents: 2500,
        taxPercentage: 10
      }).then((fixture) => {
        subscriptionId = fixture.subscriptionId;
      });
    });

    cy.then(() => {
      cy.intercept("POST", "/admin/v1/invoices/generate").as("generateInvoice");
      visitOrgPath(orgId, "/invoices/new");
      selectAutocomplete("invoice-subscription-id", subscriptionId.slice(0, 8), subscriptionId);
      cy.get("[data-testid=\"invoices-create-period-start\"]").clear().type("2026-03-01");
      cy.get("[data-testid=\"invoices-create-period-end\"]").clear().type("2026-04-01");
      cy.get("[data-testid=\"invoices-create-submit\"]").click();
    });

    cy.wait("@generateInvoice").then((interception) => {
      expect(interception.response?.body?.invoice?.total_cents).to.equal(expectedTotal);
      const invoiceId = interception.response?.body?.invoice?.id as string;
      const invoiceNumber = interception.response?.body?.invoice?.number as string;
      cy.url().should("include", `/organizations/${orgId}/invoices/${invoiceId}/manage`);
      cy.contains(".page-content", invoiceNumber).should("be.visible");
    });
  });
});
