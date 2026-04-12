describe("Admin Console Auth", () => {
  const email = Cypress.env("adminEmail") as string;
  const password = Cypress.env("adminPassword") as string;

  it("logs in and lands on dashboard", () => {
    expect(email, "CYPRESS_ADMIN_EMAIL").to.be.a("string").and.not.be.empty;
    expect(password, "CYPRESS_ADMIN_PASSWORD").to.be.a("string").and.not.be.empty;

    cy.clearCookies();
    cy.clearLocalStorage();
    cy.window().then((win) => win.sessionStorage.clear());

    cy.visit("/");

    cy.contains("Admin Login").should("be.visible");
    cy.get("[data-testid=\"auth-email\"]").type(email);
    cy.get("[data-testid=\"auth-password\"]").type(password, { log: false });
    cy.get("[data-testid=\"auth-submit\"]").click();

    cy.get("body").then(($body) => {
      if ($body.text().includes("Change Password")) {
        cy.get("[data-testid=\"auth-skip\"]").click();
      }
    });

    cy.contains("Revenue Pulse", { timeout: 10000 }).should("be.visible");
    cy.get("[data-testid=\"topbar-logout\"]").should("be.visible");
  });
});
