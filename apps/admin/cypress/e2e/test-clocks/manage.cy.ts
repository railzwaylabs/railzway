describe("Test Clock", () => {
  beforeEach(() => {
    cy.login();
  });

  it("creates, pauses, resumes, and writes audit logs", () => {
    const uid = Date.now();
    const clockName = `Clock ${uid}`;

    cy.intercept("POST", "/admin/v1/test-clocks").as("createClock");
    cy.orgVisit("/test-clock");

    cy.get("[data-testid=\"testclock-name\"]").type(clockName);
    cy.get("[data-testid=\"testclock-current-time\"]")
      .clear()
      .type("2026-03-01T00:00");
    cy.get("[data-testid=\"testclock-save\"]").click();

    cy.wait("@createClock").then((interception) => {
      const clockId = interception.response?.body?.id as string;
      expect(clockId).to.be.a("string").and.not.be.empty;
      cy.wrap(clockId).as("clockId");
    });

    cy.contains(".data-table", clockName).should("be.visible");

    cy.get("@clockId").then((clockId) => {
      cy.intercept("POST", `/admin/v1/test-clocks/${String(clockId)}/pause`).as("pauseClock");
      cy.get("[data-testid=\"testclock-pause\"]").click();
      cy.wait("@pauseClock").its("response.body.status").should("eq", "paused");
    });

    cy.get("@clockId").then((clockId) => {
      cy.intercept("POST", `/admin/v1/test-clocks/${String(clockId)}/resume`).as("resumeClock");
      cy.get("[data-testid=\"testclock-resume\"]").click();
      cy.wait("@resumeClock").its("response.body.status").should("eq", "active");
    });

    cy.intercept("GET", "/admin/v1/audit-logs*").as("auditLogs");
    cy.orgVisit("/audit-logs");
    cy.get("[data-testid=\"audit-logs-filter-action\"]").type("testclock.create");
    cy.get("[data-testid=\"audit-logs-filter-resource-type\"]").type("test_clock");
    cy.get("[data-testid=\"audit-logs-apply\"]").click();

    cy.wait("@auditLogs");
    cy.contains("testclock.create").should("exist");
  });

  it("rejects create without a valid frozen time", () => {
    cy.orgVisit("/test-clock");

    cy.get("[data-testid=\"testclock-current-time\"]").clear();
    cy.get("[data-testid=\"testclock-save\"]").click();

    cy.contains(".toast-title", "Choose a valid starting time.").should("be.visible");
  });
});
