describe("Subscriptions", () => {
  beforeEach(() => {
    cy.login();
  });

  it("create, filter, edit", () => {
    const uid = Date.now();
    const customerName = `Sub Customer ${uid}`;
    const customerEmail = `sub_${uid}@example.com`;
    const planName = `Sub Plan ${uid}`;
    const planCode = `sub_plan_${uid}`;
    const priceName = "Base Price";
    const periodStart = "2026-03-01T00:00:00Z";
    const periodEnd = "2026-04-01T00:00:00Z";

    cy.csrfRequest("POST", "/admin/v1/customers", {
      name: customerName,
      email: customerEmail,
      currency: "USD"
    })
      .then((resp) => {
        const customerId = resp.body.id as string;
        return cy.csrfRequest("POST", "/admin/v1/plans", {
          code: planCode,
          name: planName,
          active: true
        }).then((planResp) => ({ customerId, planId: planResp.body.id as string }));
      })
      .then(({ customerId, planId }) => {
        return cy.csrfRequest("POST", `/admin/v1/plans/${planId}/prices`, {
          code: `${planCode}_price`,
          name: priceName,
          price_type: "flat",
          billing_interval: "month",
          billing_interval_count: 1,
          active: true
        }).then((priceResp) => ({ customerId, planId, priceId: priceResp.body.id as string }));
      })
      .then(({ customerId, planId, priceId }) => {
        return cy.csrfRequest("POST", `/admin/v1/plans/prices/${priceId}/amounts`, {
          currency: "USD",
          unit_amount_cents: 1000
        }).then(() => ({ customerId, planId, priceId }));
      })
      .then(() => {
        const customerLabel = `${customerName} · ${customerEmail}`;
        const planLabel = `${planName} · ${planCode}`;
        const priceLabel = `${priceName} · flat · month`;
        cy.wrap({ customerLabel, planLabel, priceLabel }).as("subscriptionLabels");
      });

    cy.intercept("POST", "/admin/v1/subscriptions").as("createSubscription");
    cy.intercept("GET", "/admin/v1/customers*").as("customerOptions");
    cy.intercept("GET", "/admin/v1/plans*").as("planOptions");
    cy.orgVisit("/subscriptions/new");
    cy.contains("h2", "New Subscription").should("be.visible");

    cy.get("@subscriptionLabels").then((labels) => {
      const { customerLabel, planLabel, priceLabel } = labels as Record<string, string>;
      cy.wait("@customerOptions");
      cy.get("[data-testid=\"subscription-customer-id-list\"] option").should(($options) => {
        const values = [...$options].map((option) => option.value);
        expect(values).to.include(customerLabel);
      });
      cy.get("[data-testid=\"subscription-customer-id\"]").clear().type(customerLabel);

      cy.wait("@planOptions");
      cy.get("[data-testid=\"subscription-plan-id-list\"] option").should(($options) => {
        const values = [...$options].map((option) => option.value);
        expect(values).to.include(planLabel);
      });
      cy.get("[data-testid=\"subscription-plan-id\"]").clear().type(planLabel);

      cy.get("[data-testid=\"subscription-plan-price-id-list\"] option").should(($options) => {
        const values = [...$options].map((option) => option.value);
        expect(values).to.include(priceLabel);
      });
      cy.get("[data-testid=\"subscription-plan-price-id\"]").clear().type(priceLabel);
      cy.get("[data-testid=\"subscriptions-create-quantity\"]").clear().type("1");
      cy.get("[data-testid=\"subscriptions-create-currency\"]").clear().type("USD");
      cy.get("[data-testid=\"subscriptions-create-period-start\"]").clear().type(periodStart);
      cy.get("[data-testid=\"subscriptions-create-period-end\"]").clear().type(periodEnd);
      cy.get("[data-testid=\"subscriptions-create-submit\"]").click();
    });

    cy.wait("@createSubscription").then((interception) => {
      const subscriptionId = interception.response?.body?.id as string;
      expect(subscriptionId).to.be.a("string").and.not.be.empty;
      cy.wrap(subscriptionId).as("subscriptionId");
    });

    cy.intercept("GET", "/admin/v1/subscriptions*").as("subscriptionsList");
    cy.url().should("include", "/subscriptions");
    cy.get("@subscriptionLabels").then((labels) => {
      const { customerLabel } = labels as Record<string, string>;
      cy.get("[data-testid=\"subscriptions-filter-customer\"]").clear().type(customerLabel);
      cy.get("[data-testid=\"subscriptions-filters-apply\"]").click();
    });
    cy.wait("@subscriptionsList");

    cy.get("@subscriptionId").then((subId) => {
      const id = subId as string;
      cy.get(`[data-testid="subscriptions-manage-${id}"]`).click();
    });
    cy.contains("h2", "Subscription").should("be.visible");
    cy.get("[data-testid=\"subscriptions-edit-status\"]").clear().type("canceled");
    cy.get("[data-testid=\"subscriptions-edit-submit\"]").click();
    cy.contains(".page", "Subscription updated").should("be.visible");
  });

  it("validates required fields", () => {
    cy.orgVisit("/subscriptions/new");
    cy.get("[data-testid=\"subscriptions-create-submit\"]").should("be.disabled");
    cy.get("[data-testid=\"subscriptions-create-currency\"]").clear().type("USD");
    cy.get("[data-testid=\"subscriptions-create-submit\"]").should("be.disabled");
    cy.get("[data-testid=\"subscriptions-create-period-start\"]").type("2026-03-01T00:00:00Z");
    cy.get("[data-testid=\"subscriptions-create-period-end\"]").type("2026-04-01T00:00:00Z");
    cy.get("[data-testid=\"subscriptions-create-submit\"]").should("be.disabled");
  });
});
