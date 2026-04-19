import { selectAutocomplete, switchOrg, visitOrgPath } from "../../support/billing";

describe("Ledger Integrity", () => {
  beforeEach(() => {
    cy.login();
  });

  it("posts a balanced manual ledger transaction", () => {
    const uid = Date.now();
    let orgId = "";

    cy.createOrg(`Ledger Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    });

    cy.then(() => {
      cy.intercept("POST", "/admin/v1/ledger/transactions").as("postLedger");
      visitOrgPath(orgId, "/ledger/new");
      selectAutocomplete("ledger-create-currency", "USD", "USD");
      cy.get("[data-testid=\"ledger-create-source-type\"]").type("manual_adjustment");
      cy.get("[data-testid=\"ledger-create-source-id\"]").type(`manual_${uid}`);
      cy.get("[data-testid=\"ledger-create-occurred-at\"]").clear().type("2026-03-01T00:00");

      cy.get("[data-testid=\"ledger-entry-0-account\"]").type("1000_cash");
      cy.get("[data-testid=\"ledger-entry-0-type\"]").click();
      cy.contains("[role=\"option\"]", "Debit").click();
      cy.get("[data-testid=\"ledger-entry-0-amount\"]").clear().type("1000");
      selectAutocomplete("ledger-entry-0-currency", "USD", "USD");

      cy.get("[data-testid=\"ledger-entry-1-account\"]").type("4000_revenue");
      cy.get("[data-testid=\"ledger-entry-1-type\"]").click();
      cy.contains("[role=\"option\"]", "Credit").click();
      cy.get("[data-testid=\"ledger-entry-1-amount\"]").clear().type("1000");
      selectAutocomplete("ledger-entry-1-currency", "USD", "USD");

      cy.get("[data-testid=\"ledger-post-transaction\"]").click();
    });

    cy.wait("@postLedger").then((interception) => {
      expect(interception.response?.statusCode).to.equal(200);
      expect(interception.response?.body?.transaction?.id).to.be.a("string").and.not.be.empty;
    });

    cy.url().should("include", `/organizations/${orgId}/ledger`);
  });

  it("rejects an unbalanced manual ledger transaction", () => {
    const uid = Date.now();
    let orgId = "";

    cy.createOrg(`Ledger Error Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    });

    cy.then(() => {
      cy.intercept("POST", "/admin/v1/ledger/transactions").as("postLedger");
      visitOrgPath(orgId, "/ledger/new");
      selectAutocomplete("ledger-create-currency", "USD", "USD");
      cy.get("[data-testid=\"ledger-create-source-type\"]").type("manual_adjustment");
      cy.get("[data-testid=\"ledger-create-source-id\"]").type(`unbalanced_${uid}`);

      cy.get("[data-testid=\"ledger-entry-0-account\"]").type("1000_cash");
      cy.get("[data-testid=\"ledger-entry-0-type\"]").click();
      cy.contains("[role=\"option\"]", "Debit").click();
      cy.get("[data-testid=\"ledger-entry-0-amount\"]").clear().type("1000");
      selectAutocomplete("ledger-entry-0-currency", "USD", "USD");

      cy.get("[data-testid=\"ledger-entry-1-account\"]").type("4000_revenue");
      cy.get("[data-testid=\"ledger-entry-1-type\"]").click();
      cy.contains("[role=\"option\"]", "Credit").click();
      cy.get("[data-testid=\"ledger-entry-1-amount\"]").clear().type("900");
      selectAutocomplete("ledger-entry-1-currency", "USD", "USD");

      cy.get("[data-testid=\"ledger-post-transaction\"]").click();
    });

    cy.wait("@postLedger").then((interception) => {
      expect(interception.response?.statusCode).to.equal(400);
      expect(interception.response?.body?.error).to.equal("unbalanced_entry");
    });
    cy.contains(".toast-desc", "Ledger entries must be balanced.").should("be.visible");
    cy.url().should("include", `/organizations/${orgId}/ledger/new`);
  });
});
