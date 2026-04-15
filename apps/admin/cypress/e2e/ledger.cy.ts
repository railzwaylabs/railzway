describe("Ledger", () => {
  beforeEach(() => {
    cy.login();
  });

  it("view transactions and use filters", () => {
    cy.orgVisit("/ledger");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "Ledger Transactions");
    
    // Check filters
    cy.get("[data-testid=\"ledger-filters-apply\"]").should("be.visible");
    cy.get("[data-testid=\"ledger-filters-reset\"]").should("be.visible");
  });

  it("navigate to manual transaction (positive scenario)", () => {
    cy.orgVisit("/ledger");
    cy.get("[data-testid=\"ledger-new-button\"]").click();
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "Manual Transaction");
  });
});
