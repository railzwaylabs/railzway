describe("Features", () => {
  beforeEach(() => {
    cy.login();
  });

  it("creates a boolean feature from the feature create page", () => {
    const uid = Date.now();
    const featureName = `Feature ${uid}`;
    const featureCode = `feat_${uid}`;

    cy.intercept("POST", "/admin/v1/features").as("createFeature");

    cy.orgVisit("/features/new");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "New Feature");
    cy.get("[data-testid=\"features-create-code\"]").type(featureCode);
    cy.get("[data-testid=\"features-create-name\"]").type(featureName);
    cy.get("[data-testid=\"features-create-description\"]").type("Cypress feature description");
    cy.get("[data-testid=\"features-create-submit\"]").should("not.be.disabled").click();

    cy.wait("@createFeature").then((interception) => {
      expect(interception.response?.statusCode).to.eq(200);
    });

    cy.url().should("include", "/features");
    cy.contains(".data-table", featureName).should("be.visible");
    cy.contains(".data-table", featureCode).should("be.visible");
  });

  it("keeps submit disabled until the required create fields are filled", () => {
    cy.orgVisit("/features/new");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "New Feature");
    cy.get("[data-testid=\"features-create-submit\"]").should("be.disabled");

    cy.get("[data-testid=\"features-create-code\"]").type("feat_validate");
    cy.get("[data-testid=\"features-create-submit\"]").should("be.disabled");

    cy.get("[data-testid=\"features-create-name\"]").type("Validation Feature");
    cy.get("[data-testid=\"features-create-submit\"]").should("not.be.disabled");
  });

  it("creates a metered feature and requires picking the metered type first", () => {
    const uid = Date.now();
    const featureName = `Metered Feature ${uid}`;
    const featureCode = `meter_feat_${uid}`;
    const meterCode = `mtr_${uid}`;

    cy.createOrg(`Feature Meter Org ${uid}`).then(() => {
      return cy.csrfRequest("POST", "/admin/v1/meters", {
        code: meterCode,
        name: `Feature Meter ${uid}`,
        aggregation: "sum",
        unit: "requests",
        active: true,
      });
    }).then((resp) => {
      cy.wrap(resp.body.id as string).as("meterId");
    });

    cy.intercept("POST", "/admin/v1/features").as("createFeature");

    cy.orgVisit("/features/new");
    cy.get("[data-testid=\"features-create-code\"]").type(featureCode);
    cy.get("[data-testid=\"features-create-name\"]").type(featureName);
    cy.get("[data-testid=\"features-create-meter-id\"]").should("not.exist");

    cy.selectAutocomplete("features-create-type", "Metered", "Metered");
    cy.get("[data-testid=\"features-create-meter-id\"]").should("be.visible");

    cy.get("@meterId").then((meterId) => {
      cy.get("[data-testid=\"features-create-meter-id\"]").type(meterId as string);
    });

    cy.get("[data-testid=\"features-create-submit\"]").should("not.be.disabled").click();

    cy.wait("@createFeature").then((interception) => {
      expect(interception.response?.statusCode).to.eq(200);
    });

    cy.url().should("include", "/features");
    cy.contains(".data-table", featureName).should("be.visible");
  });
});
