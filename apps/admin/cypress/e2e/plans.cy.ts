describe("Plans", () => {
  beforeEach(() => {
    cy.login();
  });

  it("create, filter, edit", () => {
    const uid = Date.now();
    const planName = `Plan ${uid}`;
    const planCode = `plan_${uid}`;

    cy.intercept("POST", "/admin/v1/plans").as("createPlan");
    cy.orgVisit("/plans/new");
    cy.contains("h1, h2", "Create Plan").should("be.visible");
    cy.get("[data-testid=\"plans-create-name\"]").type(planName);
    cy.get("[data-testid=\"plans-create-code\"]").type(planCode);
    cy.get("[data-testid=\"plans-create-description\"]").type("Cypress plan");
    cy.get("[data-testid=\"plans-create-amount\"]").clear().type("1200");
    cy.get("[data-testid=\"plans-create-currency\"]").clear().type("USD");
    cy.get("[data-testid=\"plans-create-interval\"]").select("month");
    cy.get("[data-testid=\"plans-create-interval-count\"]").clear().type("1");
    cy.get("[data-testid=\"plans-create-submit\"]").click();

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

    const updatedName = `${planName} Updated`;
    cy.get("@planId").then((planId) => {
      cy.get(`[data-testid="plans-edit-${planId as string}"]`).click();
    });
    cy.contains("h1, h2", "Edit Plan").should("be.visible");
    cy.get("[data-testid=\"plans-edit-name\"]").clear().type(updatedName);
    cy.get("[data-testid=\"plans-edit-submit\"]").click();
    cy.contains(".page-content", updatedName).should("be.visible");
  });

  it("validates required fields", () => {
    cy.orgVisit("/plans/new");
    cy.get("[data-testid=\"plans-create-submit\"]").should("be.disabled");
    cy.get("[data-testid=\"plans-create-name\"]").type("Validation Plan");
    cy.get("[data-testid=\"plans-create-submit\"]").should("be.disabled");
    cy.get("[data-testid=\"plans-create-code\"]").type("plan_validate");
    cy.get("[data-testid=\"plans-create-submit\"]").should("not.be.disabled");
  });
});
