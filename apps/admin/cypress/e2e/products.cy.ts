describe("Products", () => {
  beforeEach(() => {
    cy.login();
  });

  it("creates a product with plans and features (complex positive scenario)", () => {
    const uid = Date.now();
    const productName = `Product ${uid}`;
    const productCode = `prod_${uid}`;
    const planName = `Default Plan ${uid}`;
    const planCode = `plan_${uid}`;

    cy.intercept("POST", "/admin/v1/products").as("createProduct");
    
    cy.orgVisit("/products/new");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "New Product");

    // Product Info
    cy.get("[data-testid=\"products-create-name\"]").type(productName);
    cy.get("[data-testid=\"products-create-code\"]").type(productCode);
    cy.get("[data-testid=\"products-create-description\"]").type("Cypress product description");

    // Plan Info (first plan is there by default)
    cy.get("[data-testid=\"plans-name-0\"]").type(planName);
    cy.get("[data-testid=\"plans-code-0\"]").type(planCode);

    // Feature selection (assuming some features exist)
    cy.get("body").then(($body) => {
        if ($body.find("[data-testid^='feature-checkbox-']").length > 0) {
            cy.get("[data-testid^='feature-checkbox-']").first().click();
        }
    });

    cy.get("[data-testid=\"products-create-submit\"]").click();

    cy.wait("@createProduct").then((interception) => {
      expect(interception.response?.statusCode).to.eq(200);
    });

    cy.url().should("include", "/products");
    cy.contains(".data-table", productName).should("be.visible");
  });

  it("validates required fields (negative scenario)", () => {
    cy.orgVisit("/products/new");
    cy.get("[data-testid=\"products-create-submit\"]").should("be.enabled"); // Button might be enabled if form is empty but backend will fail? 
    // Wait, let's check validation in ProductsCreate.tsx: const disabled = saving || !form.code || !form.name
    // Actually it uses react-hook-form.
    
    cy.get("[data-testid=\"products-create-submit\"]").click();
    cy.contains("Code is required").should("exist");
    cy.contains("Name is required").should("exist");
  });
});
