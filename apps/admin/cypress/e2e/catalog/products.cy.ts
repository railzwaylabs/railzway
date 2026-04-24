describe("Products", () => {
  beforeEach(() => {
    cy.login();
  });

  it("creates a product from the product create page", () => {
    const uid = Date.now();
    const productName = `Product ${uid}`;
    const productCode = `prod_${uid}`;
    const priceCode = `price_${uid}`;

    cy.intercept("POST", "/admin/v1/products").as("createProduct");

    cy.orgVisit("/products/new");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "New Product");

    cy.get("[data-testid=\"products-create-name\"]").type(productName);
    cy.get("[data-testid=\"products-create-code\"]").type(productCode);
    cy.get("[data-testid=\"products-create-description\"]").type("Cypress product description");
    cy.get("[data-testid=\"products-plan-0-price-0-code\"]").type(priceCode);
    cy.get("[data-testid=\"products-plan-0-price-0-amount-0\"]").clear().type("1200");

    cy.get("[data-testid=\"products-create-submit\"]").click();

    cy.wait("@createProduct").then((interception) => {
      expect(interception.response?.statusCode).to.eq(200);
    });

    cy.url().should("include", "/products");
    cy.contains(".data-table", productName).should("be.visible");
  });

  it("shows required validation on create before the user can leave the page", () => {
    cy.orgVisit("/products/new");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "New Product");

    cy.get("[data-testid=\"products-create-submit\"]").click();
    cy.get("span.text-destructive").should(($errors) => {
      expect($errors.length).to.be.gte(2);
    });
    cy.url().should("include", "/products/new");

    cy.get("[data-testid=\"products-create-code\"]").type("prod_validation");
    cy.get("[data-testid=\"products-create-submit\"]").click();
    cy.get("span.text-destructive").should(($errors) => {
      expect($errors.length).to.be.gte(1);
    });
    cy.url().should("include", "/products/new");
  });
});
