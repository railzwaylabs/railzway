describe("Usage", () => {
  beforeEach(() => {
    cy.login();
  });

  it("filters usage events", () => {
    cy.intercept("GET", "/admin/v1/usage/events*").as("listUsage");
    cy.orgVisit("/usage");

    cy.get("[data-testid=\"usage-filter-status\"]").select("rated");
    cy.get("[data-testid=\"usage-filter-recorded-from\"]").clear().type("2026-03-01T00:00:00Z");
    cy.get("[data-testid=\"usage-filters-apply\"]").click();

    cy.wait("@listUsage");
  });

  it("resets usage filters", () => {
    cy.orgVisit("/usage");
    cy.get("[data-testid=\"usage-filter-status\"]").select("rated");
    cy.get("[data-testid=\"usage-filters-reset\"]").click();
    cy.get("[data-testid=\"usage-filter-status\"]").should("have.value", "");
  });
});
