import { seedBillingFixture, selectAutocomplete, switchOrg, visitOrgPath } from "../../support/billing";

describe("Admin UI Usage Idempotency", () => {
  beforeEach(() => {
    cy.login();
  });

  it("uploads duplicate-idempotency usage CSV from UI without duplicating visible events", () => {
    const uid = Date.now();
    const duplicateKey = `usage-dup-${uid}`;
    let orgId = "";
    let customerId = "";
    let customerName = "";
    let meterCode = "";

    cy.createOrg(`UI Usage Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    });

    cy.then(() => {
      return seedBillingFixture({
        uid,
        prefix: `ui_usage_${uid}`,
        includeUsage: true
      }).then((fixture) => {
        customerId = fixture.customerId;
        customerName = fixture.customerName;
        meterCode = fixture.meterCode as string;
      });
    });

    cy.then(() => {
      const csv = [
        "meter_code,customer_id,value,recorded_at,idempotency_key",
        `${meterCode},${customerId},7,2026-03-05T00:00:00Z,${duplicateKey}`,
        `${meterCode},${customerId},7,2026-03-05T00:00:00Z,${duplicateKey}`
      ].join("\n");

      cy.intercept("POST", "/admin/v1/usage/events").as("ingestUsage");
      visitOrgPath(orgId, "/usage/new");
      cy.get("[data-testid=\"usage-upload-file\"]").selectFile({
        contents: Cypress.Buffer.from(csv),
        fileName: "duplicate-usage.csv",
        mimeType: "text/csv"
      });
      cy.get("[data-testid=\"usage-upload-submit\"]").click();
      cy.wait("@ingestUsage");
      cy.wait("@ingestUsage");
    });

    cy.then(() => {
      cy.intercept("GET", "/admin/v1/usage/events*").as("listUsage");
      visitOrgPath(orgId, "/usage");
      selectAutocomplete("usage-filter-customer", customerName);
      cy.get("[data-testid=\"usage-filters-apply\"]").click();
      cy.wait("@listUsage").then((interception) => {
        expect(interception.response?.body?.events).to.have.length(1);
      });
    });
  });
});
