describe("Admin Console Smoke", () => {
  it("loads dashboard after login", () => {
    cy.login();
    cy.orgVisit("/");
    cy.contains("Revenue Pulse", { timeout: 10000 }).should("be.visible");
    cy.get("[data-testid=\"topbar-logout\"]").should("be.visible");
  });
});
