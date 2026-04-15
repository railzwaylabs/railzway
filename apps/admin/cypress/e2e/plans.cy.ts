describe("Plans", () => {
  beforeEach(() => {
    cy.login();
  });

  it("create, filter, edit (positive scenario)", () => {
    const uid = Date.now();
    const planName = `Plan ${uid}`;
    const planCode = `plan_${uid}`;

    cy.intercept("POST", "/admin/v1/plans").as("createPlan");
    cy.orgVisit("/plans/new");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "New Plan");
    cy.get("[data-testid=\"plans-create-name\"]").type(planName);
    cy.get("[data-testid=\"plans-create-code\"]").type(planCode);
    cy.get("[data-testid=\"plans-create-description\"]").type("Cypress plan");
    cy.get("[data-testid=\"plans-create-amount\"]").clear().type("1200");
    
    // For Radix UI Select, we need to click to open and click the option
    cy.get("[data-testid=\"plans-create-interval\"]").click();
    cy.get("[role=\"option\"]").contains("Monthly").click();
    
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
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "Edit Plan");
    cy.get("[data-testid=\"plans-edit-name\"]").clear().type(updatedName);
    cy.get("[data-testid=\"plans-edit-submit\"]").click();
    cy.contains(".page-content", updatedName).should("be.visible");
  });

  it("validates required fields (negative scenario)", () => {
    cy.orgVisit("/plans/new");
    cy.get("[data-testid=\"plans-create-submit\"]").should("be.disabled");
    
    cy.get("[data-testid=\"plans-create-name\"]").type("Validation Plan");
    cy.get("[data-testid=\"plans-create-submit\"]").should("be.disabled");
    
    cy.get("[data-testid=\"plans-create-code\"]").type("plan_validate");
    cy.get("[data-testid=\"plans-create-submit\"]").should("not.be.disabled");
    
    // Test for invalid amount
    cy.get("[data-testid=\"plans-create-amount\"]").clear().type("-1");
    cy.get("[data-testid=\"plans-create-submit\"]").should("be.disabled");
    // The validation message is inline but let's check for the general error text if possible
    // In PlansCreate.tsx: if (form.unitAmountCents < 0) errors.push(t("plans_create.validation.amount_min"))
    cy.contains("Unit amount cents must be >= 0").should("exist");
  });
});
