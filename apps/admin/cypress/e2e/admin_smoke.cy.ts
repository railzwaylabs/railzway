describe("Admin Console Smoke", () => {
  it("loads dashboard after login", () => {
    cy.login();
    cy.orgVisit("/");
    cy.get("[data-testid=\"page-header-title\"]", { timeout: 10000 }).should("contain", "Revenue Pulse");
    cy.get("[data-testid=\"topbar-logout\"]").should("be.visible");
  });
});
