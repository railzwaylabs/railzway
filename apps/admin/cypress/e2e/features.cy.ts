describe("Features", () => {
  beforeEach(() => {
    cy.login();
  });

  it("create, view list, and validation (positive & negative)", () => {
    const uid = Date.now();
    const featureName = `Feature ${uid}`;
    const featureCode = `feat_${uid}`;

    cy.intercept("POST", "/admin/v1/features").as("createFeature");
    
    // 1. Validation - Negative scenario
    cy.orgVisit("/features/new");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "New Feature");
    cy.get("[data-testid=\"features-create-submit\"]").should("be.disabled");
    
    cy.get("[data-testid=\"features-create-code\"]").type(featureCode);
    cy.get("[data-testid=\"features-create-submit\"]").should("be.disabled");
    
    // 2. Creation - Positive scenario
    cy.get("[data-testid=\"features-create-name\"]").type(featureName);
    cy.get("[data-testid=\"features-create-description\"]").type("Cypress feature description");
    
    // Type is boolean by default, let's keep it
    cy.get("[data-testid=\"features-create-submit\"]").click();

    cy.wait("@createFeature").then((interception) => {
      expect(interception.response?.statusCode).to.eq(200);
    });

    cy.url().should("include", "/features");
    cy.contains(".data-table", featureName).should("be.visible");
    cy.contains(".data-table", featureCode).should("be.visible");
  });

  it("create metered feature", () => {
    const uid = Date.now();
    const featureName = `Metered Feature ${uid}`;
    const featureCode = `meter_feat_${uid}`;

    cy.orgVisit("/features/new");
    cy.get("[data-testid=\"features-create-code\"]").type(featureCode);
    cy.get("[data-testid=\"features-create-name\"]").type(featureName);
    
    // Select Metered type
    cy.get("#features-create-type").click();
    cy.get("[role=\"option\"]").contains("Metered").click();
    
    // Should show meter ID field
    cy.get("[data-testid=\"features-create-meter-id\"]").should("be.visible").type("meter_123");
    
    cy.get("[data-testid=\"features-create-submit\"]").click();
    
    cy.url().should("include", "/features");
    cy.contains(".data-table", featureName).should("be.visible");
  });
});
