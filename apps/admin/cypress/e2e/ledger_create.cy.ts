describe("Ledger Create", () => {
  beforeEach(() => {
    cy.login();
  });

  it("posts a ledger transaction", () => {
    const uid = Date.now();

    cy.intercept("POST", "/admin/v1/ledger/transactions").as("postLedger");
    cy.orgVisit("/ledger/new");

    cy.get("[data-testid=\"ledger-create-currency\"]").clear().type("USD");
    cy.get("[data-testid=\"ledger-create-source-type\"]").type("invoice");
    cy.get("[data-testid=\"ledger-create-source-id\"]").type(`inv_${uid}`);
    cy.get("[data-testid=\"ledger-create-occurred-at\"]").type("2026-03-01T00:00:00Z");

    cy.get("[data-testid=\"ledger-entry-0-account\"]").type("1000_cash");
    cy.get("[data-testid=\"ledger-entry-0-type\"]").select("debit");
    cy.get("[data-testid=\"ledger-entry-0-amount\"]").clear().type("1000");
    cy.get("[data-testid=\"ledger-entry-0-currency\"]").clear().type("USD");

    cy.get("[data-testid=\"ledger-entry-1-account\"]").type("4000_revenue");
    cy.get("[data-testid=\"ledger-entry-1-type\"]").select("credit");
    cy.get("[data-testid=\"ledger-entry-1-amount\"]").clear().type("1000");
    cy.get("[data-testid=\"ledger-entry-1-currency\"]").clear().type("USD");

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
