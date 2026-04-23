import { generateInvoice, seedBillingFixture, switchOrg, visitOrgPath } from "../../support/billing";

describe("Admin UI Deep Organization Isolation", () => {
  beforeEach(() => {
    cy.login();
  });

  it("shows seeded billing resources in org A and hides them in org B", () => {
    const uid = Date.now();
    let orgA = "";
    let orgB = "";
    let customerName = "";
    let invoiceId = "";
    let invoiceNumber = "";
    let taxRateCode = "";

    cy.createOrg(`UI Isolation A ${uid}`).then((id) => {
      orgA = id;
      return switchOrg(orgA);
    });

    cy.then(() => {
      return seedBillingFixture({
        uid,
        prefix: `ui_isolation_${uid}`,
        flatAmountCents: 2400,
        taxPercentage: 10
      }).then((fixture) => {
        customerName = fixture.customerName;
        taxRateCode = fixture.taxRateCode as string;
        return generateInvoice(fixture.subscriptionId, fixture.periodStart, fixture.periodEnd);
      }).then((invoice) => {
        invoiceId = invoice.id;
        invoiceNumber = invoice.number;
        return cy.csrfRequest("POST", `/admin/v1/invoices/${invoiceId}/open`);
      });
    });

    cy.then(() => cy.createOrg(`UI Isolation B ${uid}`)).then((id) => {
      orgB = id;
    });

    cy.then(() => {
      visitOrgPath(orgA, "/customers");
      cy.contains(".data-table", customerName).should("be.visible");
      visitOrgPath(orgA, "/invoices");
      cy.contains(".data-table", invoiceNumber).should("be.visible");
      visitOrgPath(orgA, "/taxes");
      cy.contains(".data-table", taxRateCode).should("be.visible");
    });

    cy.then(() => {
      visitOrgPath(orgB, "/customers");
      cy.contains(".data-table", customerName).should("not.exist");
      visitOrgPath(orgB, "/invoices");
      cy.contains(".data-table", invoiceNumber).should("not.exist");
      visitOrgPath(orgB, "/taxes");
      cy.contains(".data-table", taxRateCode).should("not.exist");
    });
  });
});
