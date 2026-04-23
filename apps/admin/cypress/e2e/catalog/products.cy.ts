describe("Products", () => {
  beforeEach(() => {
    cy.login();
  });

  it("creates a product with plans and features (complex positive scenario)", () => {
    const uid = Date.now();
    const productName = `Product ${uid}`;
    const productCode = `prod_${uid}`;
    const priceCode = `price_${uid}`;

    cy.intercept("POST", "/admin/v1/products").as("createProduct");
    
    cy.orgVisit("/products/new");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "New Product");

    // Product Info
    cy.get("[data-testid=\"products-create-name\"]").type(productName);
    cy.get("[data-testid=\"products-create-code\"]").type(productCode);
    cy.get("[data-testid=\"products-create-description\"]").type("Cypress product description");

    // The first plan mirrors the product identity by default; only price fields need input.
    cy.get("[data-testid=\"products-plan-0-price-0-code\"]").type(priceCode);
    cy.get("[data-testid=\"products-plan-0-price-0-amount-0\"]").clear().type("1200");

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
    
    cy.get("[data-testid=\"products-create-submit\"]").click();
    cy.contains("Required").should("exist");
  });
});
