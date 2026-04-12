describe("Customers", () => {
  beforeEach(() => {
    cy.login();
  });

  it("create, filter, edit", () => {
    const uid = Date.now();
    const name = `Cypress Customer ${uid}`;
    const email = `customer_${uid}@example.com`;
    const externalId = `ext_${uid}`;

    cy.intercept("POST", "/admin/v1/customers").as("createCustomer");
    cy.orgVisit("/customers/new");
    cy.contains("h2", "New Customer").should("be.visible");
    cy.get("[data-testid=\"customers-create-name\"]").type(name);
    cy.get("[data-testid=\"customers-create-email\"]").type(email);
    cy.get("[data-testid=\"customers-create-external-id\"]").type(externalId);
    cy.get("[data-testid=\"customers-create-currency\"]").type("USD");
    cy.get("[data-testid=\"customers-create-submit\"]").click();

    cy.wait("@createCustomer").then((interception) => {
      const customerId = interception.response?.body?.id as string;
      expect(customerId).to.be.a("string").and.not.be.empty;
      cy.wrap(customerId).as("customerId");
    });

    cy.url().should("include", "/customers");
    cy.contains(".data-table", name).should("be.visible");

    cy.get("[data-testid=\"customers-filter-name\"]").clear().type(name);
    cy.get("[data-testid=\"customers-filters-apply\"]").click();
    cy.contains(".data-table", name).should("be.visible");

    const updatedName = `${name} Updated`;
    cy.get("@customerId").then((customerId) => {
      cy.get(`[data-testid="customers-edit-${customerId as string}"]`).click();
    });
    cy.contains("h2", "Edit Customer").should("be.visible");
    cy.get("[data-testid=\"customers-edit-name\"]").clear().type(updatedName);
    cy.get("[data-testid=\"customers-edit-submit\"]").click();

    cy.url().should("include", "/customers");
    cy.contains(".data-table", updatedName).should("be.visible");
  });

  it("validates required fields", () => {
    cy.orgVisit("/customers/new");
    cy.get("[data-testid=\"customers-create-submit\"]").should("be.disabled");
    cy.get("[data-testid=\"customers-create-name\"]").type("Validation User");
    cy.get("[data-testid=\"customers-create-submit\"]").should("be.disabled");
    cy.get("[data-testid=\"customers-create-email\"]").type("valid@example.com");
    cy.get("[data-testid=\"customers-create-submit\"]").should("not.be.disabled");
  });
});
