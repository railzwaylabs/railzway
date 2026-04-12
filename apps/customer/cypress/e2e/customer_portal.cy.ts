describe("Customer Portal", () => {
  beforeEach(() => {
    cy.intercept("GET", "**/customer/v1/subscriptions*", { fixture: "subscriptions.json" }).as("subs");
    cy.intercept("GET", "**/customer/v1/invoices*", { fixture: "invoices.json" }).as("invoices");
  });

  it("loads subscriptions and invoices", () => {
    cy.visit("/?org_id=org_1&customer_id=cust_1");
    cy.wait("@subs");
    cy.wait("@invoices");
    cy.contains("plan_starter").should("be.visible");
    cy.contains("INV-200").should("be.visible");
  });
});
