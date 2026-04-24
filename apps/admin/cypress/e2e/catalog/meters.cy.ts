describe("Meters", () => {
  beforeEach(() => {
    cy.login();
  });

  it("creates, filters, and edits a meter", () => {
    const uid = Date.now();
    const code = `mtr_${uid}`;
    const name = `Meter ${uid}`;
    const updatedName = `${name} Updated`;

    cy.intercept("POST", "/admin/v1/meters").as("createMeter");

    cy.orgVisit("/meters/new");
    cy.contains("h1", "New Meter").should("be.visible");
    cy.get("[data-testid=\"meters-create-submit\"]").should("be.disabled");

    cy.get("[data-testid=\"meters-create-code\"]").type(code);
    cy.get("[data-testid=\"meters-create-name\"]").type(name);
    cy.get("[data-testid=\"meters-create-aggregation\"]").click();
    cy.get("[role=\"option\"]").contains(/^sum$/i).click();
    cy.get("[data-testid=\"meters-create-unit\"]").type("requests");
    cy.get("[data-testid=\"meters-create-submit\"]").should("not.be.disabled").click();

    cy.wait("@createMeter").then((interception) => {
      expect(interception.response?.statusCode).to.eq(200);
    });

    cy.url().should("include", "/meters/");
    cy.get("[data-testid=\"meters-edit-name\"]").clear().type(updatedName);
    cy.get("[data-testid=\"meters-edit-submit\"]").click();
    cy.contains(".page-content", updatedName).should("be.visible");

    cy.orgVisit("/meters");
    cy.get("[data-testid=\"meters-filter-code\"]").clear().type(code);
    cy.get("[data-testid=\"meters-filters-apply\"]").click();
    cy.contains(".data-table", updatedName).should("be.visible");
  });

  it("keeps submit disabled until all required create fields are filled", () => {
    cy.orgVisit("/meters/new");
    cy.get("[data-testid=\"meters-create-submit\"]").should("be.disabled");

    cy.get("[data-testid=\"meters-create-code\"]").type("mtr_validate");
    cy.get("[data-testid=\"meters-create-name\"]").type("Validation Meter");
    cy.get("[data-testid=\"meters-create-submit\"]").should("be.disabled");

    cy.get("[data-testid=\"meters-create-aggregation\"]").click();
    cy.get("[role=\"option\"]").contains(/^sum$/i).click();
    cy.get("[data-testid=\"meters-create-submit\"]").should("be.disabled");

    cy.get("[data-testid=\"meters-create-unit\"]").type("requests");
    cy.get("[data-testid=\"meters-create-submit\"]").should("not.be.disabled");
  });

  it("requires an actual change before allowing meter update", () => {
    const uid = Date.now();
    const code = `mtr_edit_${uid}`;

    cy.createOrg(`Meter Edit Org ${uid}`).then(() => {
      return cy.csrfRequest("POST", "/admin/v1/meters", {
        code,
        name: `Editable Meter ${uid}`,
        aggregation: "sum",
        unit: "requests",
        active: true,
      });
    }).then((resp) => {
      const meterId = resp.body.id as string;
      cy.orgVisit(`/meters/${meterId}/edit`);
    });

    cy.get("[data-testid=\"meters-edit-name\"]").clear();
    cy.get("[data-testid=\"meters-edit-unit\"]").clear();

    cy.get("[data-testid=\"meters-edit-aggregation\"]").click();
    cy.get("[role=\"option\"]").contains("No change").click();

    cy.get("[data-testid=\"meters-edit-active\"]").click();
    cy.get("[role=\"option\"]").contains("No change").click();

    cy.get("[data-testid=\"meters-edit-submit\"]").should("be.disabled");
  });
});
