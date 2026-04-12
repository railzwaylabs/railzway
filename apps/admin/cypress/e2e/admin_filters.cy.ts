describe("Admin Filters (Autocomplete)", () => {
  it("searches customers from subscription filters", () => {
    cy.intercept("GET", "/admin/v1/customers*").as("searchCustomers");

    cy.login();
    cy.orgVisit("/subscriptions");

    cy.get("[data-testid=\"subscriptions-filter-customer\"]").clear().type("acme", { delay: 50 });

    cy.wait("@searchCustomers");
  });

  it("searches meters from usage filters", () => {
    cy.intercept("GET", "/admin/v1/meters*").as("searchMeters");

    cy.login();
    cy.orgVisit("/usage");

    cy.get("[data-testid=\"usage-filter-meter\"]").clear().type("tokens", { delay: 50 });

    cy.wait("@searchMeters");
  });
});
