describe("Checkout", () => {
  beforeEach(() => {
    cy.intercept("GET", "**/public/invoices/*", { fixture: "invoice.json" }).as("getInvoice");
    cy.intercept("POST", "**/public/invoices/*/checkout", {
      statusCode: 200,
      body: { status: "ready", checkout_url: "https://checkout.example.com" }
    }).as("createCheckout");
  });

  it("renders invoice and allows pay now", () => {
    cy.visit("/checkout/test-token");
    cy.wait("@getInvoice");
    cy.contains("INV-123").should("be.visible");
    cy.contains("Pay now").click();
    cy.wait("@createCheckout");
  });
});
