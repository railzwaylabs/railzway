describe("Payments", () => {
  beforeEach(() => {
    cy.login();
  });

  it("filters payments", () => {
    cy.intercept("GET", "/admin/v1/payments*").as("paymentsList");
    cy.orgVisit("/payments");

    cy.get("[data-testid=\"payments-filter-status\"]").select("failed");
    cy.get("[data-testid=\"payments-filters-apply\"]").click();

    cy.wait("@paymentsList");
  });

  it("resets payment filters", () => {
    cy.orgVisit("/payments");
    cy.get("[data-testid=\"payments-filter-status\"]").select("failed");
    cy.get("[data-testid=\"payments-filters-reset\"]").click();
    cy.get("[data-testid=\"payments-filter-status\"]").should("have.value", "");
  });
});
