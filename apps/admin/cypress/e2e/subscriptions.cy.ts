describe("Subscriptions", () => {
  beforeEach(() => {
    cy.login();
  });

  it("create subscription (positive scenario)", () => {
    const uid = Date.now();
    const customerName = `Sub User ${uid}`;
    const customerEmail = `sub_user_${uid}@example.com`;
    const planName = `Pro Plan ${uid}`;
    const planCode = `plan_pro_${uid}`;

    // Pre-create data via API for reliability
    cy.ensureOrg().then((orgId) => {
      cy.csrfRequest("POST", "/admin/v1/customers", {
        name: customerName,
        email: customerEmail,
        currency: "USD"
      });

      cy.csrfRequest("POST", "/admin/v1/plans", {
        name: planName,
        code: planCode,
        active: true,
        prices: [{
          code: `price_${uid}`,
          name: "Standard Price",
          price_type: "flat",
          billing_interval: "month",
          billing_interval_count: 1,
          active: true,
          amounts: [{ currency: "USD", unit_amount_cents: 2900 }]
        }]
      });
    });

    cy.orgVisit("/subscriptions/new");
    cy.contains("h1", "New Subscription").should("be.visible");

    // Select Customer
    cy.get("[data-testid=\"subscription-customer-id\"]").click();
    cy.get("[data-testid=\"subscription-customer-id-search\"]").type(customerName);
    cy.get("[role=\"option\"]").contains(customerName).click();

    // Select Plan
    cy.get("[data-testid=\"subscription-plan-id\"]").click();
    cy.get("[data-testid=\"subscription-plan-id-search\"]").type(planName);
    cy.get("[role=\"option\"]").contains(planName).click();

    // Select Price
    cy.get("[data-testid=\"subscription-plan-price-id\"]").click();
    cy.get("[role=\"option\"]").contains("Standard Price").click();

    // Set Dates (using today for simplicity)
    const today = new Date().toISOString().split("T")[0];
    const nextMonth = new Date();
    nextMonth.setMonth(nextMonth.getMonth() + 1);
    const nextMonthStr = nextMonth.toISOString().split("T")[0];

    cy.get("[data-testid=\"subscriptions-create-period-start\"]").type(`${today}T00:00`);
    cy.get("[data-testid=\"subscriptions-create-period-end\"]").type(`${nextMonthStr}T00:00`);

    cy.get("[data-testid=\"subscriptions-create-submit\"]").click();

    cy.url().should("include", "/subscriptions");
    cy.contains(".data-table", customerName).should("be.visible");
  });

  it("validates required fields (negative scenario)", () => {
    cy.orgVisit("/subscriptions/new");
    cy.get("[data-testid=\"subscriptions-create-submit\"]").should("be.disabled");
    
    // Select customer only
    cy.get("[data-testid=\"subscription-customer-id\"]").click();
    cy.get("[data-testid=\"subscription-customer-id-search\"]").type("any");
    // Just click away to close popover if no results, but here we expect validation
    
    cy.get("[data-testid=\"subscriptions-create-submit\"]").should("be.disabled");
  });
});
