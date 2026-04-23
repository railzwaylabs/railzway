import { selectAutocomplete, switchOrg, visitOrgPath } from "../../support/billing";

describe("Catalog to Subscription Integrity", () => {
  beforeEach(() => {
    cy.login();
  });

  it("keeps plan price options aligned with the selected plan", () => {
    const uid = Date.now();
    const customerName = `Catalog Customer ${uid}`;
    const customerEmail = `catalog_${uid}@example.com`;
    const productCode = `prod_catalog_${uid}`;
    const productName = `Catalog Product ${uid}`;
    const meterCode = `api_calls_${uid}`;
    const planACode = `starter_${uid}`;
    const planAName = `Starter ${uid}`;
    const planAFlatName = `Starter Monthly ${uid}`;
    const planBCode = `growth_${uid}`;
    const planBName = `Growth ${uid}`;
    const planBFlatName = `Growth Monthly ${uid}`;
    const planBUsageName = `Growth Usage ${uid}`;
    const periodStart = "2026-03-01T00:00";
    const periodEnd = "2026-04-01T00:00";

    let orgId = "";
    let meterId = "";
    let productId = "";
    let planAId = "";
    let planBId = "";
    let planBFlatPriceId = "";

    cy.createOrg(`Catalog Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    }).then(() => {
      return cy.csrfRequest("POST", "/admin/v1/customers", {
        name: customerName,
        email: customerEmail,
        currency: "USD"
      });
    }).then(() => {
      return cy.csrfRequest("POST", "/admin/v1/products", {
        code: productCode,
        name: productName,
        description: "Catalog integrity seed",
        active: true
      });
    }).then((productResp) => {
      productId = productResp.body.id as string;
      return cy.csrfRequest("POST", "/admin/v1/meters", {
        code: meterCode,
        name: `API Calls ${uid}`,
        aggregation: "sum",
        unit: "request",
        active: true
      });
    }).then((meterResp) => {
      meterId = meterResp.body.id as string;
      return cy.csrfRequest("POST", "/admin/v1/plans", {
        code: planACode,
        name: planAName,
        description: "Starter plan",
        product_id: productId,
        active: true
      });
    }).then((planResp) => {
      planAId = planResp.body.id as string;
      return cy.csrfRequest("POST", `/admin/v1/plans/${planAId}/prices`, {
        code: `${planACode}_flat`,
        name: planAFlatName,
        price_type: "flat",
        billing_interval: "month",
        billing_interval_count: 1,
        active: true
      });
    }).then((priceResp) => {
      const planAFlatPriceId = priceResp.body.id as string;
      return cy.csrfRequest("POST", `/admin/v1/prices/${planAFlatPriceId}/amounts`, {
        currency: "USD",
        unit_amount_cents: 1900
      });
    }).then(() => {
      return cy.csrfRequest("POST", "/admin/v1/plans", {
        code: planBCode,
        name: planBName,
        description: "Growth plan",
        product_id: productId,
        active: true
      });
    }).then((planResp) => {
      planBId = planResp.body.id as string;
      return cy.csrfRequest("POST", `/admin/v1/plans/${planBId}/prices`, {
        code: `${planBCode}_flat`,
        name: planBFlatName,
        price_type: "flat",
        billing_interval: "month",
        billing_interval_count: 1,
        active: true
      });
    }).then((priceResp) => {
      planBFlatPriceId = priceResp.body.id as string;
      return cy.csrfRequest("POST", `/admin/v1/prices/${planBFlatPriceId}/amounts`, {
        currency: "USD",
        unit_amount_cents: 3900
      });
    }).then(() => {
      return cy.csrfRequest("POST", `/admin/v1/plans/${planBId}/prices`, {
        code: `${planBCode}_usage`,
        name: planBUsageName,
        price_type: "usage",
        billing_interval: "month",
        billing_interval_count: 1,
        aggregate_usage: "sum",
        billing_unit: "request",
        meter_id: meterId,
        active: true
      });
    }).then((usagePriceResp) => {
      const usagePriceId = usagePriceResp.body.id as string;
      return cy.csrfRequest("POST", `/admin/v1/prices/${usagePriceId}/amounts`, {
        currency: "USD",
        unit_amount_cents: 4
      });
    });

    cy.then(() => {
      visitOrgPath(orgId, "/subscriptions/new");
    });

    cy.then(() => selectAutocomplete("subscription-customer-id", customerName));
    cy.then(() => selectAutocomplete("subscription-plan-id", planAName));

    cy.get("[data-testid=\"subscription-plan-price-id\"]").click();
    cy.contains("[cmdk-item]", planAFlatName).should("be.visible");
    cy.contains("[cmdk-item]", planBFlatName).should("not.exist");
    cy.contains("[cmdk-item]", planAFlatName).click();

    cy.then(() => selectAutocomplete("subscription-plan-id", planBName));

    cy.get("[data-testid=\"subscription-plan-price-id\"]").click();
    cy.contains("[cmdk-item]", planAFlatName).should("not.exist");
    cy.contains("[cmdk-item]", planBFlatName).should("be.visible");
    cy.contains("[cmdk-item]", planBUsageName).should("be.visible");
    cy.contains("[cmdk-item]", planBFlatName).click();

    cy.intercept("POST", "/admin/v1/subscriptions").as("createSubscription");
    cy.get("[data-testid=\"subscriptions-create-period-start\"]").clear().type(periodStart);
    cy.get("[data-testid=\"subscriptions-create-period-end\"]").clear().type(periodEnd);
    cy.get("[data-testid=\"subscriptions-create-submit\"]").click();

    cy.wait("@createSubscription").then((interception) => {
      expect(interception.request.body.plan_id).to.equal(planBId);
      expect(interception.request.body.items).to.have.length(1);
      expect(interception.request.body.items[0].plan_price_id).to.equal(planBFlatPriceId);
      const subscriptionId = interception.response?.body?.id as string;
      expect(subscriptionId).to.be.a("string").and.not.be.empty;
      return cy.csrfRequest("GET", `/admin/v1/subscriptions/${subscriptionId}`).then((resp) => {
        expect(resp.body.plan_id).to.equal(planBId);
      });
    });

    cy.url().should("include", `/organizations/${orgId}/subscriptions`);
  });
});
