describe("Taxes", () => {
  beforeEach(() => {
    cy.login();
  });

  it("filters tax rates", () => {
    cy.intercept("GET", "/admin/v1/taxes*").as("taxesList");
    cy.orgVisit("/taxes");

    cy.get("[data-testid=\"taxes-filter-code\"]").type("VAT");
    cy.get("[data-testid=\"taxes-filters-apply\"]").click();

    cy.wait("@taxesList");
  });

  it("resets tax filters", () => {
    cy.orgVisit("/taxes");
    cy.get("[data-testid=\"taxes-filter-code\"]").type("VAT");
    cy.get("[data-testid=\"taxes-filters-reset\"]").click();
    cy.get("[data-testid=\"taxes-filter-code\"]").should("have.value", "");
  });
});
