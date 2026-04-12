describe("Ledger", () => {
  beforeEach(() => {
    cy.login();
  });

  it("filters ledger transactions", () => {
    cy.intercept("GET", "/admin/v1/ledger/transactions*").as("ledgerList");
    cy.orgVisit("/ledger");

    cy.get("[data-testid=\"ledger-filter-source-type\"]").type("invoice");
    cy.get("[data-testid=\"ledger-filters-apply\"]").click();

    cy.wait("@ledgerList");
  });

  it("resets ledger filters", () => {
    cy.orgVisit("/ledger");
    cy.get("[data-testid=\"ledger-filter-source-type\"]").type("invoice");
    cy.get("[data-testid=\"ledger-filters-reset\"]").click();
    cy.get("[data-testid=\"ledger-filter-source-type\"]").should("have.value", "");
  });
});
