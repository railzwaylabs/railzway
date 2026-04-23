describe("Usage", () => {
  beforeEach(() => {
    cy.login();
  });

  it("filters usage events (positive scenario)", () => {
    cy.intercept("GET", "/admin/v1/usage/events*").as("listUsage");
    cy.orgVisit("/usage");

    // Click to open Radix UI Select
    cy.get("[data-testid=\"usage-filter-status\"]").click();
    cy.get("[role=\"option\"]").contains("Rated").click();
    
    // Type dates
    cy.get("[data-testid=\"usage-filter-recorded-from\"]").clear().type("2026-03-01T00:00");
    cy.get("[data-testid=\"usage-filters-apply\"]").click();

    cy.wait("@listUsage").then((interception) => {
      expect(interception.request.url).to.include("status=rated");
    });
  });

  it("resets usage filters (positive scenario)", () => {
    cy.orgVisit("/usage");
    
    // Set a filter
    cy.get("[data-testid=\"usage-filter-status\"]").click();
    cy.get("[role=\"option\"]").contains("Rated").click();
    
    // Reset
    cy.get("[data-testid=\"usage-filters-reset\"]").click();
    
    // Verify it's back to "All" (empty value in state, placeholder show)
    cy.get("[data-testid=\"usage-filter-status\"]").should("contain", "All");
  });
});
