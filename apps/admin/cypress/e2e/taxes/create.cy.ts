describe("Taxes create", () => {
  beforeEach(() => {
    cy.login();
  });

  it("creates a tax rate and lists it", () => {
    const uid = Date.now();
    const code = `vat_${uid}`;
    const name = `VAT ${uid}`;

    cy.orgVisit("/taxes/new");
    cy.contains("h1, h2", "New Tax Rate").should("be.visible");
    cy.get("[data-testid=\"taxes-create-code\"]").type(code);
    cy.get("[data-testid=\"taxes-create-name\"]").type(name);
    cy.get("[data-testid=\"taxes-create-percentage\"]").clear().type("11.5");
    cy.selectRadix("taxes-create-inclusive", "Exclusive");
    cy.selectRadix("taxes-create-active", "Active");
    cy.get("[data-testid=\"taxes-create-metadata\"]").type('{\"region\":\"APAC\"}', { parseSpecialCharSequences: false });
    cy.get("[data-testid=\"taxes-create-submit\"]").click();

    cy.url().should("include", "/taxes");
    cy.get("[data-testid=\"taxes-filter-code\"]").clear().type(code);
    cy.get("[data-testid=\"taxes-filters-apply\"]").click();
    cy.contains(".data-table", code).should("be.visible");
  });

  it("validates required fields", () => {
    cy.orgVisit("/taxes/new");
    cy.get("[data-testid=\"taxes-create-submit\"]").should("be.disabled");
    cy.get("[data-testid=\"taxes-create-code\"]").type("vat_validate");
    cy.get("[data-testid=\"taxes-create-submit\"]").should("be.disabled");
    cy.get("[data-testid=\"taxes-create-name\"]").type("VAT Validate");
    cy.get("[data-testid=\"taxes-create-percentage\"]").clear().type("10");
    cy.get("[data-testid=\"taxes-create-submit\"]").should("not.be.disabled");
  });
});
