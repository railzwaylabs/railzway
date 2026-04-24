import { visitOrgPath } from "../../support/billing";

const visitInOrg = (path: string) => (
  cy.get("@orgId").then((orgId) => visitOrgPath(orgId as string, path))
)

const expectOrgUrl = (suffix: string) => (
  cy.get("@orgId").then((orgId) => {
    cy.url().should("include", `/organizations/${orgId as string}${suffix}`)
  })
)

const planFeatureRow = (planIndex: number, featureCode: string) => (
  cy.get(`[data-testid="plan-feature-row-${featureCode}"]`).eq(planIndex)
)

const configurePlanFeature = (
  planIndex: number,
  featureCode: string,
  input: { enabled?: boolean; limit?: string; unit?: string; resetPeriod?: string }
) => {
  planFeatureRow(planIndex, featureCode).within(() => {
    if (input.enabled === false) {
      cy.get(`[data-testid="plan-feature-enabled-${featureCode}"]`).uncheck({ force: true })
    } else if (input.enabled === true) {
      cy.get(`[data-testid="plan-feature-enabled-${featureCode}"]`).check({ force: true })
    }

    if (input.limit != null) {
      cy.get(`[data-testid="plan-feature-limit-${featureCode}"]`).clear().type(input.limit)
    }

    if (input.unit != null) {
      cy.get(`[data-testid="plan-feature-limit-unit-${featureCode}"]`).clear().type(input.unit)
    }

    if (input.resetPeriod) {
      cy.get(`[data-testid="plan-feature-reset-period-${featureCode}"]`).click()
    }
  })

  if (input.resetPeriod) {
    cy.get("[role=\"option\"]").contains(input.resetPeriod).click()
  }
}

const assertSavedPlanFeature = (
  featureCode: string,
  input: { enabled: boolean; limit?: string; unit?: string; resetPeriod?: string }
) => {
  cy.get(`[data-testid="plan-feature-row-${featureCode}"]`).within(() => {
    if (input.enabled) {
      cy.get(`[data-testid="plan-feature-enabled-${featureCode}"]`).should("be.checked")
    } else {
      cy.get(`[data-testid="plan-feature-enabled-${featureCode}"]`).should("not.be.checked")
    }

    if (input.limit != null) {
      cy.get(`[data-testid="plan-feature-limit-${featureCode}"]`).should("have.value", input.limit)
    }

    if (input.unit != null) {
      cy.get(`[data-testid="plan-feature-limit-unit-${featureCode}"]`).should("have.value", input.unit)
    }

    if (input.resetPeriod) {
      cy.get(`[data-testid="plan-feature-reset-period-${featureCode}"]`).should("contain", input.resetPeriod)
    }
  })
}

const openPlanByCode = (planCode: string, planName: string) => {
  visitInOrg("/plans")
  cy.get("[data-testid=\"plans-filter-code\"]").clear().type(planCode)
  cy.get("[data-testid=\"plans-filters-apply\"]").click()
  cy.contains(".data-table", planName).should("be.visible")
  cy.get('[data-testid^="plans-edit-"]').first().click()
}

describe.skip("Grafana Cloud Plan Features", () => {
  beforeEach(() => {
    cy.login()
  })

  it("creates Grafana Cloud-style product tiers and preserves plan entitlements through the admin UI", () => {
    const uid = Date.now()
    const orgName = `Grafana Cloud Org ${uid}`
    const meterCode = `logs_ingest_${uid}`
    const meterName = `Logs Ingestion ${uid}`
    const ssoCode = `sso_${uid}`
    const ssoName = `SSO ${uid}`
    const logsFeatureCode = `logs_ingestion_${uid}`
    const logsFeatureName = `Logs Ingestion ${uid}`
    const productCode = `grafana_cloud_${uid}`
    const productName = `Grafana Cloud ${uid}`
    const freePlanName = `Grafana Cloud Free ${uid}`
    const freePlanCode = `grafana_cloud_free_${uid}`
    const proPlanName = `Grafana Cloud Pro ${uid}`
    const proPlanCode = `grafana_cloud_pro_${uid}`

    cy.createOrg(orgName).as("orgId")

    cy.intercept("POST", "/admin/v1/products").as("createProduct")

    cy.csrfRequest("POST", "/admin/v1/meters", {
      code: meterCode,
      name: meterName,
      aggregation: "sum",
      unit: "gb",
      active: true,
    }).then((response) => {
      expect(response.status).to.eq(200)
      const meterId = response.body?.id as string
      expect(meterId).to.be.a("string").and.not.be.empty
      cy.wrap(meterId).as("meterId")
    })

    cy.csrfRequest("POST", "/admin/v1/features", {
      code: ssoCode,
      name: ssoName,
      description: "Single sign-on entitlement",
      feature_type: "boolean",
      active: true,
    }).then((response) => {
      expect(response.status).to.eq(200)
    })

    cy.get("@meterId").then((meterId) => {
      cy.csrfRequest("POST", "/admin/v1/features", {
        code: logsFeatureCode,
        name: logsFeatureName,
        feature_type: "metered",
        meter_id: meterId as string,
        active: true,
      }).then((response) => {
        expect(response.status).to.eq(200)
      })
    })

    visitInOrg("/features")
    cy.contains(".data-table", ssoName).should("be.visible")
    cy.contains(".data-table", logsFeatureName).should("be.visible")

    visitInOrg("/products/new")
    cy.get("[data-testid=\"products-create-code\"]").type(productCode)
    cy.get("[data-testid=\"products-create-name\"]").type(productName)
    cy.get("[data-testid=\"products-create-description\"]").type("Grafana Cloud catalog tiering")

    cy.get(`[data-testid="feature-checkbox-${ssoCode}"]`).closest("div").click()
    cy.get(`[data-testid="feature-checkbox-${logsFeatureCode}"]`).closest("div").click()

    cy.get("[data-testid=\"products-create-add-plan\"]").click()

    cy.get("[data-testid=\"plans-name-0\"]").clear().type(freePlanName)
    cy.get("[data-testid=\"plans-code-0\"]").clear().type(freePlanCode)
    cy.get("[data-testid=\"plans-name-1\"]").type(proPlanName)
    cy.get("[data-testid=\"plans-code-1\"]").type(proPlanCode)

    configurePlanFeature(0, ssoCode, { enabled: false })
    configurePlanFeature(0, logsFeatureCode, {
      limit: "50",
      unit: "GB",
      resetPeriod: "Billing Period",
    })

    configurePlanFeature(1, ssoCode, { enabled: true })
    configurePlanFeature(1, logsFeatureCode, {
      limit: "500",
      unit: "GB",
      resetPeriod: "Billing Period",
    })

    cy.get("[data-testid=\"products-create-submit\"]").should("not.be.disabled").click()

    cy.wait("@createProduct").then((interception) => {
      expect(interception.response?.statusCode).to.eq(200)
      expect(interception.request.body.feature_ids).to.have.length(2)
    })

    expectOrgUrl("/products")
    cy.contains(".data-table", productName).should("be.visible")

    openPlanByCode(freePlanCode, freePlanName)
    assertSavedPlanFeature(ssoCode, { enabled: false })
    assertSavedPlanFeature(logsFeatureCode, {
      enabled: true,
      limit: "50",
      unit: "GB",
      resetPeriod: "Billing Period",
    })

    openPlanByCode(proPlanCode, proPlanName)
    assertSavedPlanFeature(ssoCode, { enabled: true })
    assertSavedPlanFeature(logsFeatureCode, {
      enabled: true,
      limit: "500",
      unit: "GB",
      resetPeriod: "Billing Period",
    })
  })
})
