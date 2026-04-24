describe("Plans", () => {
  beforeEach(() => {
    cy.login();
  });

  it("creates, filters, and edits a flat plan", () => {
    const uid = Date.now();
    const planName = `Plan ${uid}`;
    const planCode = `plan_${uid}`;
    const updatedName = `${planName} Updated`;

    cy.intercept("POST", "/admin/v1/plans").as("createPlan");

    cy.orgVisit("/plans/new");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "Create Plan");
    cy.get("[data-testid=\"plans-create-submit\"]").should("be.disabled");

    cy.get("[data-testid=\"plans-create-name\"]").type(planName);
    cy.get("[data-testid=\"plans-create-code\"]").type(planCode);
    cy.get("[data-testid=\"plans-create-description\"]").type("Cypress plan");
    cy.get("[data-testid=\"plans-create-amount\"]").clear().type("1200");
    cy.get("[data-testid=\"plans-create-interval\"]").click();
    cy.get("[role=\"option\"]").contains("Monthly").click();
    cy.get("[data-testid=\"plans-create-submit\"]").should("not.be.disabled").click();

    cy.wait("@createPlan").then((interception) => {
      const planId = interception.response?.body?.id as string;
      expect(planId).to.be.a("string").and.not.be.empty;
      cy.wrap(planId).as("planId");
    });

    cy.url().should("include", "/plans");
    cy.contains(".data-table", planName).should("be.visible");

    cy.get("[data-testid=\"plans-filter-code\"]").clear().type(planCode);
    cy.get("[data-testid=\"plans-filters-apply\"]").click();
    cy.contains(".data-table", planName).should("be.visible");

    cy.get("@planId").then((planId) => {
      cy.get(`[data-testid="plans-edit-${String(planId)}"]`).click();
    });
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "Edit Plan");
    cy.get("[data-testid=\"plans-edit-name\"]").clear().type(updatedName);
    cy.get("[data-testid=\"plans-edit-submit\"]").click();
    cy.contains(".page-content", updatedName).should("be.visible");
  });

  it("keeps submit disabled for invalid create input", () => {
    cy.orgVisit("/plans/new");
    cy.get("[data-testid=\"plans-create-submit\"]").should("be.disabled");

    cy.get("[data-testid=\"plans-create-name\"]").type("Validation Plan");
    cy.get("[data-testid=\"plans-create-code\"]").type("plan_validate");
    cy.get("[data-testid=\"plans-create-submit\"]").should("not.be.disabled");

    cy.get("[data-testid=\"plans-create-amount\"]").clear().type("-1");
    cy.get("[data-testid=\"plans-create-submit\"]").should("be.disabled");
    cy.contains("Amount must be >= 0.").should("exist");

    cy.get("[data-testid=\"plans-create-amount\"]").clear().type("1000");
    cy.get("[data-testid=\"plans-create-interval-count\"]").clear().type("0");
    cy.get("[data-testid=\"plans-create-submit\"]").should("be.disabled");
  });

  it("requires a meter when creating a usage plan", () => {
    const uid = Date.now();
    const meterCode = `plan_meter_${uid}`;

    cy.createOrg(`Plan Usage Org ${uid}`).then(() => {
      return cy.csrfRequest("POST", "/admin/v1/meters", {
        code: meterCode,
        name: `Plan Meter ${uid}`,
        aggregation: "sum",
        unit: "requests",
        active: true,
      });
    });

    cy.orgVisit("/plans/new");
    cy.get("[data-testid=\"plans-create-name\"]").type(`Usage Plan ${uid}`);
    cy.get("[data-testid=\"plans-create-code\"]").type(`usage_plan_${uid}`);
    cy.get("[data-testid=\"plans-create-type-usage\"]").click();
    cy.get("[data-testid=\"plans-create-submit\"]").should("be.disabled");

    cy.selectAutocomplete("plan-create-meter", meterCode, meterCode);
    cy.get("[data-testid=\"plans-create-submit\"]").should("not.be.disabled");
  });
});
