describe("Settings", () => {
  beforeEach(() => {
    cy.login();
  });

  it("manages API keys (positive scenario)", () => {
    const keyName = `API Key ${Date.now()}`;

    cy.orgVisit("/settings");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "Settings");

    // Create API Key
    cy.get("[data-testid=\"settings-api-key-name\"]").type(keyName);
    cy.get("[data-testid=\"settings-api-key-scopes\"]").clear().type("usage:write");
    cy.get("[data-testid=\"settings-api-key-submit\"]").click();

    // Success check
    cy.get("[data-testid=\"settings-api-key-value\"]").should("be.visible");
    cy.get("[data-testid=\"settings-api-key-copy\"]").should("be.visible");

    // Verify in table
    cy.contains(".data-table", keyName).should("be.visible");
  });

  it("validates API key name (negative scenario)", () => {
    cy.orgVisit("/settings");
    cy.get("[data-testid=\"settings-api-key-name\"]").clear();
    cy.get("[data-testid=\"settings-api-key-submit\"]").click();
    
    // Check for toast or error message
    cy.contains("API key name is required").should("exist");
  });
});
