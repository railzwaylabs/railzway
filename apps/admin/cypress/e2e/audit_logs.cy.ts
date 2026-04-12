describe("Audit Logs", () => {
  beforeEach(() => {
    cy.login();
  });

  it("filters audit logs", () => {
    cy.intercept("GET", "/admin/v1/audit-logs*").as("auditLogs");
    cy.orgVisit("/audit-logs");

    cy.get("[data-testid=\"audit-logs-filter-action\"]").type("create");
    cy.get("[data-testid=\"audit-logs-apply\"]").click();

    cy.wait("@auditLogs");
  });

  it("resets audit log filters", () => {
    cy.orgVisit("/audit-logs");
    cy.get("[data-testid=\"audit-logs-filter-action\"]").type("create");
    cy.get("[data-testid=\"audit-logs-reset\"]").click();
    cy.get("[data-testid=\"audit-logs-filter-action\"]").should("have.value", "");
  });
});
