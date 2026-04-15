describe("Payments", () => {
  beforeEach(() => {
    cy.login();
  });

  it("view payments and use filters", () => {
    cy.orgVisit("/payments");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "Payments");
    
    // Check filters
    cy.get("[data-testid=\"payments-filters-apply\"]").should("be.visible");
    cy.get("[data-testid=\"payments-filters-reset\"]").should("be.visible");
  });

  it("filter by provider (positive scenario)", () => {
    cy.orgVisit("/payments");
    cy.get("[data-testid=\"payments-filter-provider\"]").type("stripe");
    cy.get("[data-testid=\"payments-filters-apply\"]").click();
    // Even if empty, it should not crash
    cy.get(".data-table").should("exist");
  });
});
