describe("Audit Logs", () => {
  beforeEach(() => {
    cy.login();
  });

  it("view audit logs and use filters", () => {
    cy.orgVisit("/audit-logs");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "Audit Logs");
    
    // Check filters
    cy.get("[data-testid=\"audit-logs-apply\"]").should("be.visible");
    cy.get("[data-testid=\"audit-logs-reset\"]").should("be.visible");
  });

  it("filter by action (positive scenario)", () => {
    cy.orgVisit("/audit-logs");
    cy.get("[data-testid=\"audit-logs-filter-action\"]").type("login");
    cy.get("[data-testid=\"audit-logs-apply\"]").click();
    cy.get(".data-table").should("exist");
  });
});
