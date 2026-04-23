describe("Meters", () => {
  beforeEach(() => {
    cy.login();
  });

  it("create, filter, edit (positive scenario)", () => {
    const uid = Date.now();
    const code = `mtr_${uid}`;
    const name = `Meter ${uid}`;

    cy.orgVisit("/meters/new");
    cy.contains("h1", "New Meter").should("be.visible");
    cy.get("[data-testid=\"meters-create-code\"]").type(code);
    cy.get("[data-testid=\"meters-create-name\"]").type(name);
    
    // Select Aggregation
    cy.get("[data-testid=\"meters-create-aggregation\"]").click();
    cy.get("[role=\"option\"]").contains("sum").click();
    
    cy.get("[data-testid=\"meters-create-unit\"]").type("requests");
    
    // Select Status
    cy.get("[data-testid=\"meters-create-active\"]").click();
    cy.get("[role=\"option\"]").contains("Active").click();
    
    cy.get("[data-testid=\"meters-create-submit\"]").click();

    cy.url().should("include", "/meters/");
    const updatedName = `${name} Updated`;
    cy.get("[data-testid=\"meters-edit-name\"]").clear().type(updatedName);
    cy.get("[data-testid=\"meters-edit-submit\"]").click();

    cy.orgVisit("/meters");
    cy.get("[data-testid=\"meters-filter-code\"]").clear().type(code);
    cy.get("[data-testid=\"meters-filters-apply\"]").click();
    cy.contains(".data-table", updatedName).should("be.visible");
  });

  it("validates required fields (negative scenario)", () => {
    cy.orgVisit("/meters/new");
    cy.get("[data-testid=\"meters-create-submit\"]").should("be.disabled");
    
    cy.get("[data-testid=\"meters-create-code\"]").type("mtr_validate");
    cy.get("[data-testid=\"meters-create-name\"]").type("Validation Meter");
    cy.get("[data-testid=\"meters-create-submit\"]").should("be.disabled");
    
    // Select Aggregation
    cy.get("[data-testid=\"meters-create-aggregation\"]").click();
    cy.get("[role=\"option\"]").contains("sum").click();
    
    cy.get("[data-testid=\"meters-create-unit\"]").type("requests");
    cy.get("[data-testid=\"meters-create-submit\"]").should("not.be.disabled");
  });
});
