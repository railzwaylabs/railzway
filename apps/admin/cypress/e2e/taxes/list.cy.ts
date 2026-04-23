describe("Taxes", () => {
  beforeEach(() => {
    cy.login();
  });

  it("view tax rates and use filters", () => {
    cy.orgVisit("/taxes");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "Tax Rates");
    
    // Check filters
    cy.get("[data-testid=\"taxes-filters-apply\"]").should("be.visible");
    cy.get("[data-testid=\"taxes-filters-reset\"]").should("be.visible");
  });

  it("navigate to create tax rate (positive scenario)", () => {
    cy.orgVisit("/taxes");
    cy.get("[data-testid=\"taxes-create-nav\"]").click();
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "New Tax Rate");
  });
});
