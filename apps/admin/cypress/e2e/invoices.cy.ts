describe("Invoices", () => {
  beforeEach(() => {
    cy.login();
  });

  it("generate, filter, manage", () => {
    const uid = Date.now();
    const customerName = `Inv Customer ${uid}`;
    const customerEmail = `inv_${uid}@example.com`;
    const planName = `Inv Plan ${uid}`;
    const planCode = `inv_plan_${uid}`;
    const periodStart = "2026-03-01T00:00:00Z";
    const periodEnd = "2026-04-01T00:00:00Z";

    cy.csrfRequest("POST", "/admin/v1/customers", {
      name: customerName,
      email: customerEmail,
      currency: "USD"
    })
      .then((resp) => {
        const customerId = resp.body.id;
        return cy.csrfRequest("POST", "/admin/v1/plans", {
          code: planCode,
          name: planName,
          active: true
        }).then((planResp) => ({ customerId, planId: planResp.body.id }));
      })
      .then(({ customerId, planId }) => {
        return cy
          .csrfRequest("POST", `/admin/v1/plans/${planId}/prices`, {
            code: `${planCode}_price`,
            name: "Base Price",
            price_type: "flat",
            billing_interval: "month",
            billing_interval_count: 1,
            active: true
          })
          .then((priceResp) => ({ customerId, planId, priceId: priceResp.body.id }));
      })
      .then(({ customerId, planId, priceId }) => {
        return cy
          .csrfRequest("POST", `/admin/v1/plans/prices/${priceId}/amounts`, {
            currency: "USD",
            unit_amount_cents: 1000
          })
          .then(() => ({ customerId, planId, priceId }));
      })
      .then(({ customerId, planId, priceId }) => {
        return cy.csrfRequest("POST", "/admin/v1/subscriptions", {
          customer_id: customerId,
          plan_id: planId,
          currency: "USD",
          current_period_start: periodStart,
          current_period_end: periodEnd,
          items: [{ plan_price_id: priceId, quantity: 1 }]
        });
      })
      .then((resp) => {
        const subscriptionId = resp.body.id as string;
        cy.wrap(subscriptionId).as("invoiceSubscriptionId");
      });

    cy.intercept("GET", "/admin/v1/subscriptions*").as("subscriptionsList");
    cy.intercept("POST", "/admin/v1/invoices/generate").as("generateInvoice");
    cy.orgVisit("/invoices/new");
    cy.contains("h1, h2", "Generate Invoice").should("be.visible");
    cy.get("@invoiceSubscriptionId").then((subId) => {
      cy.wait("@subscriptionsList");
      cy.get("[data-testid=\"invoice-subscription-id-list\"] option").should(($options) => {
        const values = [...$options].map((option) => option.value);
        expect(values).to.include(subId as string);
      });
      cy.get("[data-testid=\"invoice-subscription-id\"]").clear().type(subId as string);
      cy.get("[data-testid=\"invoices-create-period-start\"]").clear().type(periodStart);
      cy.get("[data-testid=\"invoices-create-period-end\"]").clear().type(periodEnd);
      cy.get("[data-testid=\"invoices-create-submit\"]").click();
    });

    cy.wait("@generateInvoice").then((interception) => {
      const invoiceId = interception.response?.body?.id as string;
      const invoiceNumber = interception.response?.body?.number as string;
      expect(invoiceId).to.be.a("string").and.not.be.empty;
      expect(invoiceNumber).to.be.a("string").and.not.be.empty;
      cy.wrap({ invoiceId, invoiceNumber }).as("invoiceInfo");
    });

    cy.url().should("include", "/invoices/");
    cy.get("[data-testid=\"invoices-manage-open\"]").click();
    cy.contains(".page-content", "Manage Invoice").should("be.visible");

    cy.orgVisit("/invoices");
    cy.get("@invoiceInfo").then((info) => {
      const { invoiceNumber, invoiceId } = info as Record<string, string>;
      cy.get("[data-testid=\"invoices-filter-number\"]").clear().type(invoiceNumber);
      cy.get("[data-testid=\"invoices-filters-apply\"]").click();
      cy.contains(".data-table", invoiceNumber).should("be.visible");
      cy.get(`[data-testid="invoices-manage-${invoiceId}"]`).click();
    });
    cy.get("[data-testid=\"invoices-manage-mark-paid\"]").click();
  });

  it("validates required fields", () => {
    cy.orgVisit("/invoices/new");
    cy.get("[data-testid=\"invoices-create-submit\"]").should("be.disabled");
    cy.get("[data-testid=\"invoices-create-period-start\"]").type("2026-03-01T00:00:00Z");
    cy.get("[data-testid=\"invoices-create-submit\"]").should("be.disabled");
    cy.get("[data-testid=\"invoices-create-period-end\"]").type("2026-04-01T00:00:00Z");
    cy.get("[data-testid=\"invoices-create-submit\"]").should("be.disabled");
  });
});
