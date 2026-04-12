describe("Organization isolation", () => {
  beforeEach(() => {
    cy.login();
  });

  it("new org starts with empty core resources", () => {
    cy.createOrg().then(() => {
      cy.orgVisit("/taxes");
      cy.contains("No tax rates found").should("be.visible");

      cy.orgVisit("/meters");
      cy.contains("No meters found").should("be.visible");

      cy.orgVisit("/customers");
      cy.contains("No customers found").should("be.visible");
    });
  });
});
