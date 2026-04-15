describe("Product → Plans → Customer → Subscription flow", () => {
  beforeEach(() => {
    cy.login();
  });

  it("creates a real-world product setup with multiple prices and subscribes a customer", () => {
    const uid = Date.now();
    const productCode = `prod_${uid}`;
    const productName = `Product ${uid}`;
    const meterCode = `api_calls_${uid}`;
    const meterName = `API Calls ${uid}`;
    const starterPlanCode = `starter_${uid}`;
    const starterPlanName = `Starter ${uid}`;
    const proPlanCode = `pro_${uid}`;
    const proPlanName = `Pro ${uid}`;
    const customerName = `Customer ${uid}`;
    const customerEmail = `cust_${uid}@example.com`;
    const periodStart = "2026-03-01T00:00:00Z";
    const periodEnd = "2026-04-01T00:00:00Z";

    let productId = "";
    let meterId = "";
    let starterPlanId = "";
    let proPlanId = "";
    let basePriceId = "";
    let usagePriceId = "";
    let tieredPriceId = "";
    let customerId = "";

    cy.csrfRequest("POST", "/admin/v1/products", {
      code: productCode,
      name: productName,
      description: "Cypress product",
      active: true,
    })
      .then((resp) => {
        productId = resp.body.id as string;
        expect(productId).to.be.a("string").and.not.be.empty;
        return cy.csrfRequest("POST", "/admin/v1/meters", {
          code: meterCode,
          name: meterName,
          aggregation: "sum",
          unit: "request",
          active: true,
        });
      })
      .then((resp) => {
        meterId = resp.body.id as string;
        expect(meterId).to.be.a("string").and.not.be.empty;
        return cy.csrfRequest("POST", "/admin/v1/plans", {
          code: starterPlanCode,
          name: starterPlanName,
          description: "Starter plan",
          product_id: productId,
          active: true,
        });
      })
      .then((resp) => {
        starterPlanId = resp.body.id as string;
        expect(starterPlanId).to.be.a("string").and.not.be.empty;
        return cy.csrfRequest("POST", "/admin/v1/plans", {
          code: proPlanCode,
          name: proPlanName,
          description: "Pro plan with hybrid pricing",
          product_id: productId,
          active: true,
        });
      })
      .then((resp) => {
        proPlanId = resp.body.id as string;
        expect(proPlanId).to.be.a("string").and.not.be.empty;
        return cy.csrfRequest("POST", `/admin/v1/plans/${proPlanId}/prices`, {
          code: `${proPlanCode}_base`,
          name: "Base Subscription",
          price_type: "flat",
          billing_interval: "month",
          billing_interval_count: 1,
          active: true,
        });
      })
      .then((resp) => {
        basePriceId = resp.body.id as string;
        expect(basePriceId).to.be.a("string").and.not.be.empty;
        return cy.csrfRequest("POST", `/admin/v1/prices/${basePriceId}/amounts`, {
          currency: "USD",
          unit_amount_cents: 2900,
        });
      })
      .then(() => {
        return cy.csrfRequest("POST", `/admin/v1/plans/${proPlanId}/prices`, {
          code: `${proPlanCode}_usage`,
          name: "API Usage",
          price_type: "usage",
          billing_interval: "month",
          billing_interval_count: 1,
          aggregate_usage: "sum",
          billing_unit: "request",
          meter_id: meterId,
          active: true,
        });
      })
      .then((resp) => {
        usagePriceId = resp.body.id as string;
        expect(usagePriceId).to.be.a("string").and.not.be.empty;
        return cy.csrfRequest("POST", `/admin/v1/prices/${usagePriceId}/amounts`, {
          currency: "USD",
          unit_amount_cents: 5,
        });
      })
      .then(() => {
        return cy.csrfRequest("POST", `/admin/v1/plans/${proPlanId}/prices`, {
          code: `${proPlanCode}_tiered`,
          name: "Tiered Usage",
          price_type: "tiered",
          billing_interval: "month",
          billing_interval_count: 1,
          aggregate_usage: "sum",
          billing_unit: "request",
          meter_id: meterId,
          active: true,
        });
      })
      .then((resp) => {
        tieredPriceId = resp.body.id as string;
        expect(tieredPriceId).to.be.a("string").and.not.be.empty;
        return cy.csrfRequest("POST", `/admin/v1/prices/${tieredPriceId}/tiers`, {
          tier_mode: "graduated",
          start_quantity: 0,
          end_quantity: 1000,
          unit_amount_cents: 0,
          unit: "request",
        });
      })
      .then(() => {
        return cy.csrfRequest("POST", `/admin/v1/prices/${tieredPriceId}/tiers`, {
          tier_mode: "graduated",
          start_quantity: 1000,
          unit_amount_cents: 2,
          unit: "request",
        });
      })
      .then(() => {
        return cy.csrfRequest("POST", "/admin/v1/customers", {
          name: customerName,
          email: customerEmail,
          currency: "USD",
        });
      })
      .then((resp) => {
        customerId = resp.body.id as string;
        expect(customerId).to.be.a("string").and.not.be.empty;
        return cy.csrfRequest("POST", "/admin/v1/subscriptions", {
          customer_id: customerId,
          plan_id: proPlanId,
          currency: "USD",
          current_period_start: periodStart,
          current_period_end: periodEnd,
          items: [
            { plan_price_id: basePriceId, quantity: 1 },
            { plan_price_id: usagePriceId, quantity: 1 },
            { plan_price_id: tieredPriceId, quantity: 1 },
          ],
        });
      })
      .then((resp) => {
        const subscriptionId = resp.body.id as string;
        expect(subscriptionId).to.be.a("string").and.not.be.empty;
      });

    cy.orgVisit("/products");
    cy.contains(".data-table", productName).should("be.visible");

    cy.orgVisit("/plans");
    cy.get("[data-testid=\"plans-filter-code\"]").clear().type(proPlanCode);
    cy.get("[data-testid=\"plans-filters-apply\"]").click();
    cy.contains(".data-table", proPlanName).should("be.visible");

    cy.orgVisit("/subscriptions");
    cy.contains(".data-table", customerName).should("be.visible");
  });
});
