describe("Invoice app", () => {
  it("renders sample invoice", () => {
    cy.visit("/");
    cy.contains("INV-90823").should("be.visible");
    cy.contains("Acme Solutions Inc.").should("be.visible");
  });
});
