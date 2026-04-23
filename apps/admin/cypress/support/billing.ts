const ORG_KEY = "railzway_admin_org_id";

export type BillingFixture = {
  uid: number;
  productId: string;
  productName: string;
  productCode: string;
  planId: string;
  planName: string;
  planCode: string;
  flatPriceId: string;
  flatPriceName: string;
  flatAmountCents: number;
  customerId: string;
  customerName: string;
  customerEmail: string;
  subscriptionId: string;
  periodStart: string;
  periodEnd: string;
  taxRateId?: string;
  taxRateCode?: string;
  taxPercentage?: number;
  meterId?: string;
  meterCode?: string;
  usagePriceId?: string;
  usagePriceName?: string;
  usageUnitAmountCents?: number;
};

export type SeedBillingFixtureOptions = {
  uid?: number;
  prefix?: string;
  flatAmountCents?: number;
  taxPercentage?: number;
  includeUsage?: boolean;
  usageUnitAmountCents?: number;
  periodStart?: string;
  periodEnd?: string;
};

export type GeneratedInvoice = {
  id: string;
  number: string;
  subtotal_cents: number;
  tax_cents: number;
  total_cents: number;
  amount_due_cents: number;
  amount_paid_cents: number;
  status: string;
};

export function switchOrg(orgId: string): Cypress.Chainable<string> {
  Cypress.env("orgId", orgId);
  return cy.csrfRequest({
    method: "POST",
    url: `/admin/v1/auth/using/${orgId}`
  }).then(() => cy.wrap(orgId));
}

export function visitOrgPath(orgId: string, path: string): Cypress.Chainable<Window> {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return switchOrg(orgId).then(() => {
    return cy.visit(`/organizations/${orgId}${normalized}`, {
      onBeforeLoad(win) {
        win.localStorage.setItem(ORG_KEY, orgId);
      }
    });
  });
}

export function selectAutocomplete(id: string, query: string, optionText?: string): Cypress.Chainable<void> {
  return cy.selectAutocomplete(id, query, optionText);
}

export function seedBillingFixture(options: SeedBillingFixtureOptions = {}): Cypress.Chainable<BillingFixture> {
  const uid = options.uid ?? Date.now();
  const prefix = options.prefix ?? `billing_${uid}`;
  const periodStart = options.periodStart ?? "2026-03-01T00:00:00Z";
  const periodEnd = options.periodEnd ?? "2026-04-01T00:00:00Z";
  const flatAmountCents = options.flatAmountCents ?? 2500;
  const taxPercentage = options.taxPercentage;
  const includeUsage = options.includeUsage ?? false;
  const usageUnitAmountCents = options.usageUnitAmountCents ?? 125;

  const fixture: Partial<BillingFixture> = {
    uid,
    productName: `Product ${prefix}`,
    productCode: `prod_${prefix}`,
    planName: `Plan ${prefix}`,
    planCode: `plan_${prefix}`,
    flatPriceName: `Base ${prefix}`,
    flatAmountCents,
    customerName: `Customer ${prefix}`,
    customerEmail: `${prefix}@example.com`,
    periodStart,
    periodEnd,
    taxPercentage
  };

  if (includeUsage) {
    fixture.meterCode = `meter_${prefix}`;
    fixture.usagePriceName = `Usage ${prefix}`;
    fixture.usageUnitAmountCents = usageUnitAmountCents;
  }
  if (taxPercentage != null) {
    fixture.taxRateCode = `tax_${prefix}`;
  }

  return cy.csrfRequest("POST", "/admin/v1/products", {
    code: fixture.productCode,
    name: fixture.productName,
    description: `Seed product ${prefix}`,
    active: true
  }).then((productResp) => {
    fixture.productId = productResp.body.id as string;
    return cy.csrfRequest("POST", "/admin/v1/plans", {
      code: fixture.planCode,
      name: fixture.planName,
      description: `Seed plan ${prefix}`,
      product_id: fixture.productId,
      active: true
    });
  }).then((planResp) => {
    fixture.planId = planResp.body.id as string;
    return cy.csrfRequest("POST", `/admin/v1/plans/${fixture.planId}/prices`, {
      code: `${fixture.planCode}_flat`,
      name: fixture.flatPriceName,
      price_type: "flat",
      billing_interval: "month",
      billing_interval_count: 1,
      active: true
    });
  }).then((flatPriceResp) => {
    fixture.flatPriceId = flatPriceResp.body.id as string;
    return cy.csrfRequest("POST", `/admin/v1/prices/${fixture.flatPriceId}/amounts`, {
      currency: "USD",
      unit_amount_cents: flatAmountCents
    });
  }).then(() => {
    if (!includeUsage) {
      return null;
    }
    return cy.csrfRequest("POST", "/admin/v1/meters", {
      code: fixture.meterCode,
      name: `Meter ${prefix}`,
      aggregation: "sum",
      unit: "request",
      active: true
    }).then((meterResp) => {
      fixture.meterId = meterResp.body.id as string;
      return cy.csrfRequest("POST", `/admin/v1/plans/${fixture.planId}/prices`, {
        code: `${fixture.planCode}_usage`,
        name: fixture.usagePriceName,
        price_type: "usage",
        billing_interval: "month",
        billing_interval_count: 1,
        aggregate_usage: "sum",
        billing_unit: "request",
        meter_id: fixture.meterId,
        active: true
      });
    }).then((usagePriceResp) => {
      fixture.usagePriceId = usagePriceResp.body.id as string;
      return cy.csrfRequest("POST", `/admin/v1/prices/${fixture.usagePriceId}/amounts`, {
        currency: "USD",
        unit_amount_cents: usageUnitAmountCents
      });
    });
  }).then(() => {
    if (taxPercentage == null) {
      return null;
    }
    return cy.csrfRequest("POST", "/admin/v1/taxes", {
      code: fixture.taxRateCode,
      name: `Tax ${prefix}`,
      percentage: taxPercentage,
      inclusive: false,
      active: true
    }).then((taxResp) => {
      fixture.taxRateId = taxResp.body.id as string;
    });
  }).then(() => {
    return cy.csrfRequest("POST", "/admin/v1/customers", {
      name: fixture.customerName,
      email: fixture.customerEmail,
      currency: "USD"
    });
  }).then((customerResp) => {
    fixture.customerId = customerResp.body.id as string;
    const items: Array<{ plan_price_id: string; quantity: number }> = [
      { plan_price_id: fixture.flatPriceId as string, quantity: 1 }
    ];
    if (fixture.usagePriceId) {
      items.push({ plan_price_id: fixture.usagePriceId, quantity: 1 });
    }
    return cy.csrfRequest("POST", "/admin/v1/subscriptions", {
      customer_id: fixture.customerId,
      plan_id: fixture.planId,
      currency: "USD",
      current_period_start: periodStart,
      current_period_end: periodEnd,
      items
    });
  }).then((subscriptionResp) => {
    fixture.subscriptionId = subscriptionResp.body.id as string;
    return cy.wrap(fixture as BillingFixture);
  });
}

export function generateInvoice(
  subscriptionId: string,
  periodStart: string,
  periodEnd: string,
  issueAt?: string,
  dueAt?: string
): Cypress.Chainable<GeneratedInvoice> {
  return cy.csrfRequest("POST", "/admin/v1/invoices/generate", {
    subscription_id: subscriptionId,
    period_start: periodStart,
    period_end: periodEnd,
    issue_at: issueAt,
    due_at: dueAt
  }).then((resp) => cy.wrap(resp.body.invoice as GeneratedInvoice));
}
