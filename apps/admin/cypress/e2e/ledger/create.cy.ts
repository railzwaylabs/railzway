describe("Ledger Create", () => {
  beforeEach(() => {
    cy.login();
  });

  it("posts a ledger transaction", () => {
    const sourceId = crypto.randomUUID();

    cy.intercept("POST", "/admin/v1/ledger/transactions").as("postLedger");
    cy.orgVisit("/ledger/new");

    cy.selectAutocomplete("ledger-create-currency", "USD", "USD");
    cy.get("[data-testid=\"ledger-create-source-type\"]").type("adjustment");
    cy.get("[data-testid=\"ledger-create-source-id\"]").type(sourceId);
    cy.get("[data-testid=\"ledger-create-occurred-at\"]").type("2026-03-01T00:00");

    cy.get("[data-testid=\"ledger-entry-0-account\"]").type("1000_cash");
    cy.selectRadix("ledger-entry-0-type", "Debit");
    cy.get("[data-testid=\"ledger-entry-0-amount\"]").clear().type("1000");
    cy.selectAutocomplete("ledger-entry-0-currency", "USD", "USD");

    cy.get("[data-testid=\"ledger-entry-1-account\"]").type("4000_revenue");
    cy.selectRadix("ledger-entry-1-type", "Credit");
    cy.get("[data-testid=\"ledger-entry-1-amount\"]").clear().type("1000");
    cy.selectAutocomplete("ledger-entry-1-currency", "USD", "USD");

    cy.get("[data-testid=\"ledger-post-transaction\"]").click();
    cy.wait("@postLedger");
    cy.url().should("include", "/ledger");
  });

  it("rejects empty entries", () => {
    cy.orgVisit("/ledger/new");
    cy.get("[data-testid=\"ledger-post-transaction\"]").click();
    cy.contains(".toast-title", "At least 2 valid entries required").should("be.visible");
  });
});
