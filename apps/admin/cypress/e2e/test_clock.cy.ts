describe("Test Clock", () => {
  beforeEach(() => {
    cy.login();
  });

  it("writes audit logs on upsert", () => {
    cy.intercept("POST", "/admin/v1/test-clock").as("saveClock");
    cy.orgVisit("/test-clock");

    cy.get("[data-testid=\"testclock-current-time\"]")
      .clear()
      .type("2026-03-01T00:00:00Z");
    cy.get("[data-testid=\"testclock-status\"]").select("active");
    cy.get("[data-testid=\"testclock-save\"]").click();

    cy.wait("@saveClock");

    cy.intercept("GET", "/admin/v1/audit-logs*").as("auditLogs");
    cy.orgVisit("/audit-logs");
    cy.get("[data-testid=\"audit-logs-filter-action\"]").type("testclock.upsert");
    cy.get("[data-testid=\"audit-logs-filter-resource-type\"]").type("test_clock");
    cy.get("[data-testid=\"audit-logs-apply\"]").click();

    cy.wait("@auditLogs");
    cy.contains("testclock.upsert").should("exist");
  });
});
